package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func dashify(name string) string {
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func generateUniqueFilename(baseName string, mediaColl *mongo.Collection) string {
	newFilename := baseName
	counter := 2
	for {
		var result bson.M
		err := mediaColl.FindOne(context.Background(), bson.M{"filename": newFilename}).Decode(&result)
		if err == mongo.ErrNoDocuments {
			break
		}
		newFilename = fmt.Sprintf("%s-%d", baseName, counter)
		counter++
	}
	return newFilename
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	mongoURI := os.Getenv("MONGO_DB_URI")
	dbName := os.Getenv("MONGO_DB_NAME")
	mediaCollection := os.Getenv("MONGO_DB_COLLECTION")
	bucketName := os.Getenv("AWS_BUCKET")
	region := os.Getenv("AWS_REGION")
	endpoint := os.Getenv("AWS_ENDPOINT")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	mediaID := os.Getenv("MEDIA_ID")
	imageUpdateLimit := 100

	if accessKeyID == "" || secretAccessKey == "" {
		log.Fatalf("AWS credentials not found. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables.")
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database(dbName)
	mediaColl := db.Collection(mediaCollection)

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}
	s3Client := s3.New(sess)

	var filter bson.M
	if mediaID != "" {
		objectID, err := primitive.ObjectIDFromHex(mediaID)
		if err != nil {
			log.Fatalf("Invalid media ID format: %v", err)
		}
		filter = bson.M{"_id": objectID}
		fmt.Println("Finding specific media by ID:", mediaID)
	} else {
		filter = bson.M{
			"$or": []bson.M{
				{"filename": bson.M{"$regex": `\s`}},
				{"filename": bson.M{"$regex": `_`}},
				{"filename": bson.M{"$regex": `--`}},
			},
		}
		fmt.Println("Processing non-dash-separated filenames...")
	}

	cursor, err := mediaColl.Find(context.Background(), filter, options.Find().SetLimit(int64(imageUpdateLimit)))
	if err != nil {
		log.Fatalf("Failed to query media collection: %v", err)
	}

	for cursor.Next(context.Background()) {
		var media bson.M
		if err := cursor.Decode(&media); err != nil {
			log.Fatalf("Failed to decode media document: %v", err)
		}

		filename := media["filename"].(string)
		fmt.Printf("Retrieved filename from MongoDB: %s\n", filename)

		baseFilename := dashify(filename)
		newFilename := generateUniqueFilename(baseFilename, mediaColl)

		oldKey := filename
		newKey := newFilename

		_, err := s3Client.CopyObject(&s3.CopyObjectInput{
			Bucket:     aws.String(bucketName),
			CopySource: aws.String(fmt.Sprintf("%s/%s", bucketName, oldKey)),
			Key:        aws.String(newKey),
		})
		if err != nil {
			log.Fatalf("Failed to rename object in R2: %v", err)
		}

		_, err = s3Client.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(oldKey),
		})
		if err != nil {
			log.Fatalf("Failed to delete old object in R2: %v", err)
		}

		filter := bson.M{"_id": media["_id"]}
		update := bson.M{"$set": bson.M{"filename": newFilename}}
		_, err = mediaColl.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Fatalf("Failed to update media collection: %v", err)
		}

		fmt.Printf("Renamed %s to %s in R2 and MongoDB\n", oldKey, newKey)
	}
}
