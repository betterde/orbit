package mongodb

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/event"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/betterde/orbit/internal/journal"
)

type Option struct {
	Key   interface{} `json:"key"`
	Label string      `json:"label"`
}

type Dao interface {
	getAttribute(name string) (interface{}, error)
}

var (
	Client   *mongo.Client
	Database *mongo.Database
)

func Init(ctx context.Context) {
	var err error

	currentCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

	defer cancel()

	uri := viper.GetString("database.mongodb.uri")
	if uri == "" {
		journal.Logger.Panicw("You must set your 'ORBIT_DATABASE_MONGODB_URI' environmental variable or 'database.mongodb.uri' config.")
	}

	logMonitor := event.CommandMonitor{
		Started: func(ctx context.Context, startedEvent *event.CommandStartedEvent) {
			journal.Logger.Infof("mongo reqId:%d start on db:%s cmd:%s sql:%+v", startedEvent.RequestID, startedEvent.DatabaseName,
				startedEvent.CommandName, startedEvent.Command)
		},
		Succeeded: func(ctx context.Context, succeededEvent *event.CommandSucceededEvent) {
			journal.Logger.Infof("mongo reqId:%d exec cmd:%s success duration %d ms", succeededEvent.RequestID,
				succeededEvent.CommandName, succeededEvent.Duration.Milliseconds())
		},
		Failed: func(ctx context.Context, failedEvent *event.CommandFailedEvent) {
			journal.Logger.Errorf("mongo reqId:%d exec cmd:%s failed duration %d ms", failedEvent.RequestID,
				failedEvent.CommandName, failedEvent.Duration.Milliseconds())
		},
	}

	//loggerOptions := options.Logger().SetComponentLevel(options.LogComponentAll, options.LogLevelDebug)

	clientOptions := options.Client().ApplyURI(uri).SetMonitor(&logMonitor)
	Client, err = mongo.Connect(currentCtx, clientOptions)
	if err != nil {
		journal.Logger.Panicw("Unable to establish connection to MongoDB server!", err)
	}

	err = Client.Ping(currentCtx, readpref.Primary())
	if err != nil {
		journal.Logger.Panicw("An error occurred while communicating with the MongoDB server!", err)
	}
}

func SetDatabase(name string) *mongo.Database {
	if Client != nil {
		Database = Client.Database(name)
	}

	return Database
}

// ConvertToOptions Convert dao list to options
func ConvertToOptions(key, label string, coll []Dao) (options []Option, err error) {
	for _, item := range coll {
		value, err := item.getAttribute(key)
		if err != nil {
			return nil, err
		}

		text, err := item.getAttribute(label)
		if err != nil {
			return nil, err
		}

		options = append(options, Option{
			Key:   value,
			Label: fmt.Sprintf("%v", text),
		})
	}

	return options, nil
}
