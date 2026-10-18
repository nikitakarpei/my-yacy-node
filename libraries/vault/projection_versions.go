package vault

const projectionVersionsBucket Name = "vault_projection_versions"

var projectionVersionKeys = SingleKey(TextKeyPartFrom(
	func(bucket Name) string { return string(bucket) },
	func(text string) (Name, error) { return Name(text), nil },
)).KeyLayout()

type projectionVersionCodec struct{}

func (projectionVersionCodec) Encode(version string) ([]byte, error) {
	return []byte(version), nil
}

func (projectionVersionCodec) Decode(stored []byte) (string, error) {
	return string(stored), nil
}

type projectionVersions struct {
	stored *Collection[Name, string]
}

func (v *Vault) registerProjectionVersions() (projectionVersions, error) {
	stored, err := v.RegisterCollection(
		projectionVersionsBucket,
		projectionVersionKeys,
		projectionVersionCodec{},
	)

	return projectionVersions{stored: stored}, err
}

func (p projectionVersions) staleAmong(
	tx *Txn,
	buckets []projectionBucket,
) ([]projectionBucket, error) {
	var stale []projectionBucket
	for _, bucket := range buckets {
		storedVersion, found, err := p.stored.Get(tx, bucket.bucketName())
		if err != nil {
			return nil, err
		}
		if !found || storedVersion != bucket.registeredVersion() {
			stale = append(stale, bucket)
		}
	}

	return stale, nil
}

func (p projectionVersions) record(tx *Txn, buckets []projectionBucket) error {
	for _, bucket := range buckets {
		if _, err := p.stored.Put(tx, bucket.bucketName(), bucket.registeredVersion()); err != nil {
			return err
		}
	}

	return nil
}
