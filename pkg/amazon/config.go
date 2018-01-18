package amazon

// Config ...
type Config struct {
	AccessKeyID     string // aws access key id
	SecretAccessKey string // aws secret access key
	Region          string // aws region
	Bucket          string // aws s3 bucket name to upload the file into
	Timeout         string // timeot, time to do the upload within
	Format          string // format of the disk being uploaded
}
