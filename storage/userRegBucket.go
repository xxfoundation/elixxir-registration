package storage

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/rateLimiting"
	"io/ioutil"
	"os"
)

// Read in a user registration bucket from a path
func ReadUserRegBucket(userRegBucketPath string) (*rateLimiting.Bucket, error) {
	bucketData, err := ioutil.ReadFile(userRegBucketPath)
	var bucket rateLimiting.Bucket
	if err != nil {
		return nil, errors.WithMessage(err, "ReadUserRegBucket: file read error")
	} else {
		// Try deserializing the bucket data
		err := json.Unmarshal(bucketData, &bucket)
		if err != nil {
			return nil, errors.WithMessage(err, "ReadUserRegBucket: deserialization error")
		}
	}
	return &bucket, nil
}

func WriteUserRegBucket(bucket *rateLimiting.Bucket, bucketPath string) error {
	bucketState, err := json.Marshal(bucket)
	if err == nil {
		// write bucket state to file
		if bucketPath != "" {
			err = ioutil.WriteFile(bucketPath, bucketState, os.FileMode(0664))
			if err != nil {
				return errors.WithMessage(err, "WriteUserRegBucket: file write error")
			}
		}
	} else {
		return errors.WithMessage(err, "WriteUserRegBucket: file write error")
	}
	return nil
}
