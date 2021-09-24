package main

import (
	"context"
	"log"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Conn ...class object of mongo connection
type Conn struct {
	Client *mongo.Client
}
type MongoField struct {
	Key   string
	Value string
}

func parseURI(shards string) string {
	cwd, _ := os.Getwd()
	path := ""
	// check if it's a Windows or Linux URI
	if strings.Contains(cwd, "\\") {
		path = cwd + "\\mongocert.pem"
	} else {
		path = "/etc/ssl/certs/mongocert.pem"
	}
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("Could not find the cert.pem file i the CWD, nor in the /etc/ssl/certs directory")
		panic("Please copy a certificate file from mongoDB into the CWD and rename it to mongocert.pem")
	} else {
		uri := "mongodb://" + shards + "/Dashboard?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&tlsCertificateKeyFile=" + path
		return uri
	}
}

func NewConn(shards string) (*Conn, error) {
	URI := parseURI(shards)
	clientOptions := options.Client().ApplyURI(URI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, err
	}
	return &Conn{client}, nil
}

// InsertPost ...(dbName, collection, row) into the DB
func (x Conn) InsertPost(db string, coll string, row interface{}, ctx context.Context) {
	client := x.Client
	// ref := reflect.ValueOf(row)
	// for i := 0; i < ref.NumField(); i++ {
	// 	bsonPost = append(bsonPost, bson.E{Key: ref.Type().Field(i).Name, Value: ref.Field(i).String()})
	// }
	collection := client.Database(db).Collection(coll)
	log.Println("posting into database: " + db + " collection: " + coll)
	insertResult, err := collection.InsertOne(ctx, row)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("insertion complete!", insertResult.InsertedID)
}

func (x Conn) GetCollection(db string, coll string, ctx context.Context) ([]interface{}, error) {
	client := x.Client
	collection := client.Database(db).Collection(coll)
	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var count int
	var lines []interface{}
	for cursor.Next(ctx) {
		count = count + 1
		var res interface{}
		err := cursor.Decode(&res)
		lines = append(lines, res)
		if err != nil {
			return nil, err
		}
	}
	return lines, nil
}

func (x Conn) ReplaceEntry(db string, coll string, filter interface{}, replacement interface{}, ctx context.Context) error {
	client := x.Client
	collection := client.Database(db).Collection(coll)
	_, err := collection.ReplaceOne(ctx, filter, replacement)
	if err != nil {
		return err
	}
	log.Println("Replaced successfully")
	return nil
}

func (x Conn) DeleteOne(db string, coll string, filter interface{}, ctx context.Context) error {
	client := x.Client
	collection := client.Database(db).Collection(coll)
	_, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	log.Println("Deleted successfully")
	return nil
}
