package microsoft

// Config ...
type Config struct {
	StorageAccount string // Name of the storage account
	StorageKey     string // Access key of storage account. There are two keys under access keys incase one needs to be revoked.
	Container      string // Name of the existing container or container to be created.
}
