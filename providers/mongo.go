package providers

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"

	"github.com/Kozical/taskengine/job"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MongoConfig struct {
	Addrs          []string `json:"addrs"`
	Port           int      `json:"port"`
	User           string   `json:"user"`
	Pass           string   `json:"pass"`
	UseTLS         bool     `json:"use_tls"`
	UseInsecureTLS bool     `json:"use_insecure_tls"`
	CAPath         string   `json:"ca_path"`
}

type MongoFindAction struct {
	Database   string            `json:"Database"`
	Collection string            `json:"Collection"`
	Query      map[string]string `json:"Query"`
	Limit      string            `json:"Limit"`
	Sort       string            `json:"Sort"`
}

type MongoState struct {
	Result string
}

func (d MongoState) GetProperty(property string) string {
	if property == "Result" {
		return d.Result
	}
	return ""
}

var session *mgo.Session

func init() {
	f, err := os.Open("config/mongo.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var config MongoConfig
	d := json.NewDecoder(f)

	err = d.Decode(&config)
	if err != nil {
		panic(err)
	}

	info := &mgo.DialInfo{
		Addrs:    config.Addrs,
		Username: config.User,
		Password: config.Pass,
		FailFast: true,
	}
	if config.UseTLS {
		var tlsConfig *tls.Config
		if config.UseInsecureTLS {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			pool := x509.NewCertPool()
			b, err := ioutil.ReadFile(config.CAPath)
			if err != nil {
				panic(err)
			}
			ok := pool.AppendCertsFromPEM(b)
			if !ok {
				panic(errors.New("Failed to read certificates from CAPath"))
			}
			tlsConfig = &tls.Config{
				RootCAs: pool,
			}
		}
		info.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), tlsConfig)
		}
	}
	fmt.Printf("Dialing mongo %s:%d\n", config.Addrs, config.Port)

	session, err = mgo.DialWithInfo(info)
	if err != nil {
		panic(err)
	}

	fmt.Println("Registering Mongo")
	job.RegisterActionProvider("mongo_find_action", MongoFindActionFunc)
}

func MongoFindActionFunc(raw json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	var settings MongoFindAction

	err = json.Unmarshal(raw, &settings)
	if err != nil {
		return
	}

	if len(settings.Database) == 0 {
		err = errors.New("Database parameter not provided to Datastore")
		return
	}
	if len(settings.Database) == 0 {
		err = errors.New("Collection parameter not provided to Datastore")
		return
	}

	var query interface{}

	if len(settings.Query) == 0 {
		query = nil
	} else {
		query = settings.Query
	}

	q := session.DB(settings.Database).C(settings.Collection).Find(query)

	if len(settings.Limit) > 0 {
		var i int
		i, err = strconv.Atoi(settings.Limit)
		if err != nil {
			return
		}
		q = q.Limit(i)
	}

	if len(settings.Sort) > 0 {
		q = q.Sort(settings.Sort)
	}

	var result []bson.M
	q.All(&result)

	b, err := json.Marshal(&result)
	if err != nil {
		return
	}
	s = MongoState{
		Result: string(b),
	}
	return
}
