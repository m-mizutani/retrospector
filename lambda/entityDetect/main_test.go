package main_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/m-mizutani/retrospector"
	"github.com/m-mizutani/retrospector/pkg/lambda"
	"github.com/m-mizutani/retrospector/pkg/mock"
	"github.com/m-mizutani/retrospector/pkg/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	main "github.com/m-mizutani/retrospector/lambda/entityDetect"
)

func TestEntityDetect(t *testing.T) {
	// Setup test event
	sqsEvent, err := createSQSEvent("us-east-5", events.S3Entity{
		Bucket: events.S3Bucket{
			Name: "blue",
		},
		Object: events.S3Object{
			Key: "my/entity-matched",
		},
	})
	require.NoError(t, err)

	newS3, _ := mock.NewS3Mock()
	s3Svc := service.NewEntityService(newS3)
	wq := s3Svc.NewWriteQueue("us-east-5", "blue", "my/entity-matched")
	wq.Write(&retrospector.Entity{
		Value: retrospector.Value{
			Data: "five",
			Type: retrospector.ValueDomainName,
		},
		Source:     "timeless",
		RecordedAt: time.Now().Unix(),
	})
	require.NoError(t, wq.Close())

	t.Run("matched", func(t *testing.T) {
		// Setup mock
		repo := mock.NewRepository()
		repo.PutIOCSet([]*retrospector.IOC{
			{
				Value: retrospector.Value{
					Data: "five",
					Type: retrospector.ValueDomainName,
				},
			},
		})
		httpClient := &mock.HTTPClient{
			RespCode: http.StatusOK,
			RespBody: ioutil.NopCloser(strings.NewReader("")),
		}
		args := &lambda.Arguments{
			Repository:      repo,
			Event:           sqsEvent,
			NewS3:           newS3,
			HTTP:            httpClient,
			SlackWebhookURL: "https://test.example.com/slack",
		}
		require.NoError(t, main.Handler(args))

		require.Equal(t, 1, len(httpClient.Requests))
		assert.Equal(t, "test.example.com", httpClient.Requests[0].URL.Host)
		assert.Equal(t, "/slack", httpClient.Requests[0].URL.Path)
	})

	t.Run("mismatched by data", func(t *testing.T) {
		// Setup mock
		repo := mock.NewRepository()
		repo.PutIOCSet([]*retrospector.IOC{
			{
				Value: retrospector.Value{
					Data: "six",
					Type: retrospector.ValueDomainName,
				},
			},
		})
		httpClient := &mock.HTTPClient{
			RespCode: http.StatusOK,
			RespBody: ioutil.NopCloser(strings.NewReader("")),
		}
		args := &lambda.Arguments{
			Repository:      repo,
			Event:           sqsEvent,
			NewS3:           newS3,
			HTTP:            httpClient,
			SlackWebhookURL: "https://test.example.com/slack",
		}
		require.NoError(t, main.Handler(args))
		assert.Equal(t, 0, len(httpClient.Requests))
	})

	t.Run("mismatched by type", func(t *testing.T) {
		// Setup mock
		repo := mock.NewRepository()
		repo.PutIOCSet([]*retrospector.IOC{
			{
				Value: retrospector.Value{
					Data: "five",
					Type: retrospector.ValueIPAddr,
				},
			},
		})
		httpClient := &mock.HTTPClient{
			RespCode: http.StatusOK,
			RespBody: ioutil.NopCloser(strings.NewReader("")),
		}
		args := &lambda.Arguments{
			Repository:      repo,
			Event:           sqsEvent,
			NewS3:           newS3,
			HTTP:            httpClient,
			SlackWebhookURL: "https://test.example.com/slack",
		}
		require.NoError(t, main.Handler(args))
		assert.Equal(t, 0, len(httpClient.Requests))
	})
}

func createSQSEvent(region string, s3entities events.S3Entity) (*events.SQSEvent, error) {
	event := events.S3Event{
		Records: []events.S3EventRecord{
			{
				AWSRegion: region,
				S3:        s3entities,
			},
		},
	}
	rawEvent, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	snsEntity := events.SNSEntity{
		Message: string(rawEvent),
	}
	rawSNSEntity, err := json.Marshal(snsEntity)
	if err != nil {
		return nil, err
	}

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				Body: string(rawSNSEntity),
			},
		},
	}

	return &sqsEvent, nil
}