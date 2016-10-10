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

type MongoState struct {
	Result string
}

func (d MongoState) GetProperty(property string) string {
	if property == "Result" {
		return d.Result
	}
	return ""
}

type MongoActionProvider struct {
	configPath string
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
	Session *mgo.Session
}

func NewMongoActionProvider(path string) (ap *MongoActionProvider, err error) {
	ap = &MongoActionProvider{
		configPath: path,
	}
	err = ap.init()
	return
}

func (ap *MongoActionProvider) Name() string {
	return "mongo_action"
}

func (ap *MongoActionProvider) Cleanup() {
	ap.Session.Close()
}

func (ap *MongoActionProvider) init() (err error) {
	var f *os.File
	f, err = os.Open("config/mongo.json")
	if err != nil {
		return
	}
	defer f.Close()

	d := json.NewDecoder(f)

	err = d.Decode(&ap.Config)
	if err != nil {
		return
	}

	info := &mgo.DialInfo{
		Addrs:    ap.Config.Addrs,
		Username: ap.Config.User,
		Password: ap.Config.Pass,
		FailFast: true,
	}
	if ap.Config.UseTLS {
		var tlsConfig *tls.Config
		if ap.Config.UseInsecureTLS {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			var b []byte
			pool := x509.NewCertPool()
			b, err = ioutil.ReadFile(ap.Config.CAPath)
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
	ap.Session, err = mgo.DialWithInfo(info)
	return
}

func (ap *MongoActionProvider) Action(raw json.RawMessage, j *job.Job) (s job.StateObject, err error) {
	err = json.Unmarshal(raw, &ap.Settings)
	if err != nil {
		return
	}

	if len(ap.Settings.Database) == 0 {
		err = errors.New("Database parameter not provided to Datastore")
		return
	}
	if len(ap.Settings.Database) == 0 {
		err = errors.New("Collection parameter not provided to Datastore")
		return
	}

	var query interface{}

	if len(ap.Settings.Query) == 0 {
		query = nil
	} else if len(ap.Settings.ObjectID) > 0 {
		query = bson.M{"_id": bson.ObjectIdHex(ap.Settings.ObjectID)}
	} else {
		query = ap.Settings.Query
	}

	q := ap.Session.DB(ap.Settings.Database).C(ap.Settings.Collection).Find(query)

	if len(ap.Settings.Limit) > 0 {
		var i int
		i, err = strconv.Atoi(ap.Settings.Limit)
		if err != nil {
			return
		}
		q = q.Limit(i)
	}

	if len(ap.Settings.Sort) > 0 {
		q = q.Sort(ap.Settings.Sort)
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
