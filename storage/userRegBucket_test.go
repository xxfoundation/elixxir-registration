package storage

import (
	"gitlab.com/elixxir/primitives/rateLimiting"
	"os"
	"reflect"
	"testing"
)

// Read and write a bucket to a file.
// It should be equivalent after having existed in the file
func TestReadWriteUserRegBucket(t *testing.T) {
	_, err := os.Stat(".test_files")
	if os.IsNotExist(err) {
		err = os.Mkdir(".test_files", os.FileMode(0775))
		if err != nil {
			t.Fatal(err)
		}
	}
	bucket := rateLimiting.CreateBucketFromLeakRatio(100, 0.5, nil)
	err = WriteUserRegBucket(bucket, ".test_files/userRegBucket.json")
	if err != nil {
		t.Error(err)
	}
	var readBucket *rateLimiting.Bucket
	readBucket, err = ReadUserRegBucket(".test_files/userRegBucket.json")
	if !reflect.DeepEqual(bucket, readBucket) {
		t.Error("Bucket wasn't the same after reading and writing")
	}
}
