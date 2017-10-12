package main

import (
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	sess            *session.Session
	bucket          = aws.String(os.Getenv("AWS_BUCKET"))
	bucketURLPrefix = "https://" + os.Getenv("AWS_BUCKET") + ".s3.amazonaws.com/"
)

func initS3() {
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
}

func loadMahua() (mahuas []string) {
	svc := s3.New(sess)
	resp, err := svc.ListObjects(&s3.ListObjectsInput{Bucket: bucket})

	if err != nil {
		log.Printf("Unable to list buckets, %v", err)
	}
	for _, item := range resp.Contents {
		if keys := strings.Split(*item.Key, "/"); len(keys) == 2 && keys[0] == "mahua" && !strings.HasSuffix(keys[1], "_thumbnail.jpg") {
			mahuas = append(mahuas, *item.Key)
		}
	}
	log.Printf("%d mahua pics loaded\n", len(mahuas))
	return mahuas
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
