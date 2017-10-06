package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	sess   *session.Session
	bucket = aws.String(os.Getenv("AWS_BUCKET"))
)

func initS3() {
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := s3.New(sess)
	resp, err := svc.ListObjects(&s3.ListObjectsInput{Bucket: bucket})

	if err != nil {
		log.Printf("Unable to list buckets, %v", err)
	}
	mahuas := []string{}
	for _, item := range resp.Contents {
		fmt.Println("Name:         ", *item.Key)
		fmt.Println("Last modified:", *item.LastModified)
		fmt.Println("Size:         ", *item.Size)
		fmt.Println("Storage class:", *item.StorageClass)
		fmt.Println("")
		if strings.Split(*item.Key, "/")[0] == "mahua" {
			mahuas = append(mahuas, item.Key)
		}
	}
}

func upload(filename, localDIR, s3DIR string, tmp bool) (string, error) {
	file, err := os.Open(localDIR + "/" + filename)

	if err != nil {
		return "", err
	}

	defer func() {
		if tmp {
			err := os.Remove(localDIR + "/" + filename)
			if err != nil {
				log.Println(err)
			}
		}
	}()
	defer file.Close()

	uploader := s3manager.NewUploader(sess)
	res, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      bucket,
		ContentType: aws.String("image/jpeg"),
		ACL:         aws.String("public-read"),
		Key:         aws.String(s3DIR + "/" + filename),
		Body:        file,
	})
	return res.Location, err
}
