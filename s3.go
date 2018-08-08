package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	minio "github.com/minio/minio-go"
)

var (
	bucketURLBase   = os.Getenv("AWS_BUCKET_URL_BASE")
	endpoint        = os.Getenv("AWS_ENDPOINT")
	accessKeyID     = os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucket          = os.Getenv("AWS_BUCKET")
	client          *minio.Client
)

func initS3() {
	fmt.Printf("%+v\n", strings.Join(os.Environ(), "\n"))
	_client, err := minio.NewV2(endpoint, accessKeyID, secretAccessKey, true)
	if err != nil {
		log.Fatalln(err)
	}
	client = _client
}

func loadMahua() (mahuas []string) {
	// Create a done channel to control 'ListObjects' go routine.
	doneCh := make(chan struct{})

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)
	for item := range client.ListObjects(bucket, "", true, doneCh) {
		if item.Err != nil {
			fmt.Println(item.Err)
			return
		}
		if !strings.HasSuffix(item.Key, "_thumbnail.jpg") {
			mahuas = append(mahuas, item.Key)
		}
	}
	log.Printf("%d mahua pics loaded\n", len(mahuas))
	return
}

func upload(filename, localDIR string, tmp bool) (string, error) {
	filePath := localDIR + "/" + filename

	defer func() {
		if tmp {
			err := os.Remove(localDIR + "/" + filename)
			if err != nil {
				log.Println(err)
			}
		}
	}()

	_, err := client.FPutObject(bucket, filename, filePath, minio.PutObjectOptions{ContentType: "image/jpeg", UserMetadata: map[string]string{"x-amz-acl": "public-read"}})

	return bucketURLBase + filename, err
}
