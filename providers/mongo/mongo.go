package mongo

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

	"github.com/Kozical/taskengine/core/runner"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var session *mgo.Session

/*
type Provider interface {
	Execute(*Job) (StateObject, error)
	Register(*Job, json.RawMessage) error
	New() Provider
	Name() string
	Cleanup()
}
*/

type MongoProvider struct {
	Properties map[string]string
	Config     struct {
		Addrs          []string `json:"addrs"`
		Port           int      `json:"port"`
		User           string   `json:"user"`
		Pass           string   `json:"pass"`
		UseTLS         bool     `json:"use_tls"`
		UseInsecureTLS bool     `json:"use_insecure_tls"`
		CAPath         string   `json:"ca_path"`
	}
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

	err = d.Decode(&mp.Config)
	if err != nil {
		return
	}

	info := &mgo.DialInfo{
		Addrs:    mp.Config.Addrs,
		Username: mp.Config.User,
		Password: mp.Config.Pass,
		FailFast: true,
	}

	if mp.Config.UseTLS {
		var tlsConfig *tls.Config
		if mp.Config.UseInsecureTLS {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			var b []byte
			pool := x509.NewCertPool()
			b, err = ioutil.ReadFile(mp.Config.CAPath)
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

func (mp *MongoProvider) Execute(j *runner.Job) (err error) {
	var task *runner.Task
	for _, t := range j.Tasks {
		if t.Provider == mp {
			task = &t
			break
		}
	}
	if task == nil {
		err = errors.New("MongoProvider received a nil task")
		return
	}

	for _, name := range []string{"Result"} {
		mp.Properties[name] = fmt.Sprintf("%s.%s", task.Title, name)
	}

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

	j.State[mp.Properties["Result"]] = func() interface{} { return string(b) }
	return
}
