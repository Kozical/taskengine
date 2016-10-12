package mongo

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"strconv"

	"github.com/Kozical/taskengine/job"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var config MongoConfig
var session *mgo.Session

type MongoConfig struct {
	Addrs          []string `json:"addrs"`
	Port           int      `json:"port"`
	User           string   `json:"user"`
	Pass           string   `json:"pass"`
	UseTLS         bool     `json:"use_tls"`
	UseInsecureTLS bool     `json:"use_insecure_tls"`
	CAPath         string   `json:"ca_path"`
}

/*
type Provider interface {
	Execute(*Job) (StateObject, error)
	Register(*Job, json.RawMessage) error
	New() Provider
	Name() string
	Cleanup()
}
*/

type MongoState struct {
	Result string
}

func (m MongoState) GetProperty(property string) interface{} {
	if property == "Result" {
		return m.Result
	}
	return ""
}

type MongoProvider struct {
	Settings struct {
		Database   string            `json:"Database"`
		Collection string            `json:"Collection"`
		Query      map[string]string `json:"Query"`
		Limit      string            `json:"Limit"`
		Sort       string            `json:"Sort"`
		ObjectID   string            `json:"ObjectId"`
	}
}

func NewMongoProvider(path string) (mp *MongoProvider, err error) {
	mp = new(MongoProvider)

	var f *os.File
	f, err = os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	d := json.NewDecoder(f)

	err = d.Decode(&config)
	if err != nil {
		return
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
			var b []byte
			pool := x509.NewCertPool()
			b, err = ioutil.ReadFile(config.CAPath)
			if err != nil {
				return
			}
			ok := pool.AppendCertsFromPEM(b)
			if !ok {
				err = errors.New("Failed to read certificates from CAPath")
				return
			}
			tlsConfig = &tls.Config{
				RootCAs: pool,
			}
		}
		info.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), tlsConfig)
		}
	}
	session, err = mgo.DialWithInfo(info)
	return
}

func (mp *MongoProvider) Name() string {
	return "mongo"
}

func (mp *MongoProvider) Cleanup() {
	if session != nil {
		session.Close()
		session = nil
	}
}

func (mp *MongoProvider) New() job.Provider {
	return &MongoProvider{}
}

func (mp *MongoProvider) Register(j *job.Job, raw json.RawMessage) (err error) {
	err = json.Unmarshal(raw, &mp.Settings)
	if err != nil {
		return
	}

	if len(mp.Settings.Database) == 0 {
		err = errors.New("Database parameter not provided to Datastore")
		return
	}
	if len(mp.Settings.Collection) == 0 {
		err = errors.New("Collection parameter not provided to Datastore")
		return
	}
	return
}

func (mp *MongoProvider) Execute(j *job.Job) (s job.StateObject, err error) {
	var query interface{}

	if len(mp.Settings.Query) == 0 {
		query = nil
	} else if len(mp.Settings.ObjectID) > 0 {
		query = bson.M{"_id": bson.ObjectIdHex(mp.Settings.ObjectID)}
	} else {
		query = mp.Settings.Query
	}

	q := session.DB(mp.Settings.Database).C(mp.Settings.Collection).Find(query)

	if len(mp.Settings.Limit) > 0 {
		var i int
		i, err = strconv.Atoi(mp.Settings.Limit)
		if err != nil {
			return
		}
		q = q.Limit(i)
	}

	if len(mp.Settings.Sort) > 0 {
		q = q.Sort(mp.Settings.Sort)
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
