package yacyproto

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	seedColHash                = "Hash"
	seedColName                = "Name"
	seedColPeerType            = "PeerType"
	seedColRemotePeerType      = "yourtype"
	seedColPrimaryAddress      = "IP"
	seedColAdditionalAddresses = "IP6"
	seedColPort                = "Port"
	seedColSecurePort          = "PortSSL"
	seedColSeedListAddress     = "seedURL"
	seedColCapabilities        = "Flags"
	seedColVersion             = "Version"
	seedColTags                = "Tags"
	seedColSolrAvailable       = "SorlAvail"
	seedColLastSeen            = "LastSeen"
	seedColFirstSeen           = "BDate"
	seedColDisconnectedAt      = "dct"
	seedColUTCOffset           = "UTC"
	seedColUptime              = "Uptime"
	seedColIndexingSpeed       = "ISpeed"
	seedColRetrievalSpeed      = "RSpeed"
	seedColUplinkSpeed         = "USpeed"
	seedColClientConnectRate   = "CCount"
	seedColIndexedWords        = "ICount"
	seedColStoredURLs          = "LCount"
	seedColNoticedURLs         = "NCount"
	seedColRemoteCrawlURLs     = "RCount"
	seedColStoredSeeds         = "SCount"
	seedColWordsSent           = "sI"
	seedColWordsReceived       = "rI"
	seedColURLsSent            = "sU"
	seedColURLsReceived        = "rU"
	seedColNews                = "news"
)

const (
	seedAddressSeparator = "|"
	seedSolrAvailableYes = "OK"
	seedSolrAvailableNo  = "NA"

	// seedMaxPlainBytes bounds a decoded seed so a dropped unknown key can't
	// turn it into unbounded storage.
	seedMaxPlainBytes = 4096
)

var seedColumns = map[string]bool{
	seedColHash: true, seedColName: true, seedColPeerType: true,
	seedColRemotePeerType: true, seedColPrimaryAddress: true,
	seedColAdditionalAddresses: true, seedColPort: true, seedColSecurePort: true,
	seedColSeedListAddress: true, seedColCapabilities: true, seedColVersion: true,
	seedColTags: true, seedColSolrAvailable: true, seedColLastSeen: true,
	seedColFirstSeen: true, seedColDisconnectedAt: true, seedColUTCOffset: true,
	seedColUptime: true, seedColIndexingSpeed: true, seedColRetrievalSpeed: true,
	seedColUplinkSpeed: true, seedColClientConnectRate: true,
	seedColIndexedWords: true, seedColStoredURLs: true, seedColNoticedURLs: true,
	seedColRemoteCrawlURLs: true, seedColStoredSeeds: true, seedColWordsSent: true,
	seedColWordsReceived: true, seedColURLsSent: true, seedColURLsReceived: true,
	seedColNews: true,
}

// seedWireCodec translates between the seed domain type and YaCy's peer
// property-form line. It is the only thing outside this file that knows the
// wire schema and the base64 framing that carries a seed between peers.
type seedWireCodec struct{}

func (c seedWireCodec) decode(ctx context.Context, framed string) (yacymodel.Seed, error) {
	plain, err := decodeWireForm(ctx, framed)
	if err != nil {
		return yacymodel.Seed{}, fmt.Errorf("%w: %w", yacymodel.ErrBadSeed, err)
	}
	if len(plain) > seedMaxPlainBytes {
		return yacymodel.Seed{}, fmt.Errorf(
			"%w: seed %d bytes exceeds %d", yacymodel.ErrBadSeed, len(plain), seedMaxPlainBytes,
		)
	}

	if open := strings.IndexByte(plain, propertyOpen); open >= 0 {
		plain = plain[open+1:]
	}
	if end := strings.LastIndexByte(plain, propertyClose); end >= 0 {
		plain = plain[:end]
	}

	properties, err := parsePropertyPairs(plain)
	if err != nil {
		return yacymodel.Seed{}, fmt.Errorf("%w: %w", yacymodel.ErrBadSeed, err)
	}
	for key := range properties {
		if !seedColumns[key] {
			delete(properties, key)
		}
	}

	return seedWireForm{properties: properties}.domain(ctx)
}

func (c seedWireCodec) decodeRemote(ctx context.Context, framed string) (yacymodel.Seed, error) {
	seed, err := c.decode(ctx, framed)
	if err != nil {
		return yacymodel.Seed{}, err
	}
	if !seed.IsAddressable() {
		return yacymodel.Seed{}, fmt.Errorf("%w: remote seed has no address", yacymodel.ErrBadSeed)
	}
	return seed, nil
}

func (c seedWireCodec) encode(seed yacymodel.Seed) string {
	return seedWireFormFromDomain(seed).framed()
}

// seedWireForm is the seed in its property-form wire representation: the flat,
// protocol-defined column map YaCy exchanges between peers.
type seedWireForm struct {
	properties map[string]string
	columns    []string
}

func (f seedWireForm) domain(ctx context.Context) (yacymodel.Seed, error) {
	hash, err := yacymodel.ParseHash(f.properties[seedColHash])
	if err != nil {
		return yacymodel.Seed{}, fmt.Errorf("%w: hash: %w", yacymodel.ErrBadSeed, err)
	}
	name, err := yacymodel.ParsePeerName(f.properties[seedColName])
	if err != nil {
		return yacymodel.Seed{}, fmt.Errorf("%w: name: %w", yacymodel.ErrBadSeed, err)
	}
	peerType, err := yacymodel.ParsePeerType(f.properties[seedColPeerType])
	if err != nil {
		return yacymodel.Seed{}, fmt.Errorf("%w: peer type: %w", yacymodel.ErrBadSeed, err)
	}

	seed := yacymodel.Seed{
		Hash:              hash,
		Name:              name,
		PeerType:          peerType,
		PrimaryAddress:    f.host(seedColPrimaryAddress),
		Port:              f.port(seedColPort),
		SecurePort:        f.port(seedColSecurePort),
		RemotePeerType:    f.peerType(seedColRemotePeerType),
		Version:           f.version(),
		SolrAvailable:     f.solrAvailable(),
		FirstSeen:         f.timestamp(seedColFirstSeen),
		LastSeen:          f.timestamp(seedColLastSeen),
		DisconnectedAt:    f.epochMillis(seedColDisconnectedAt),
		UTCOffset:         f.utcOffset(),
		Uptime:            time.Duration(f.integer(seedColUptime)) * time.Minute,
		IndexingSpeed:     f.integer(seedColIndexingSpeed),
		RetrievalSpeed:    f.integer(seedColRetrievalSpeed),
		UplinkSpeed:       f.integer(seedColUplinkSpeed),
		ClientConnectRate: f.decimal(seedColClientConnectRate),
		IndexedWords:      f.integer(seedColIndexedWords),
		StoredURLs:        f.integer(seedColStoredURLs),
		NoticedURLs:       f.integer(seedColNoticedURLs),
		RemoteCrawlURLs:   f.integer(seedColRemoteCrawlURLs),
		StoredSeeds:       f.integer(seedColStoredSeeds),
		WordsSent:         f.integer(seedColWordsSent),
		WordsReceived:     f.integer(seedColWordsReceived),
		URLsSent:          f.integer(seedColURLsSent),
		URLsReceived:      f.integer(seedColURLsReceived),
	}

	seed.AdditionalAddresses = f.additionalAddresses()

	if capabilities, ok := f.properties[seedColCapabilities]; ok {
		seed.Capabilities = yacymodel.Some(peerCapabilitiesWireCodec{}.decode(capabilities))
	}

	tags, err := peerTagsWireCodec{}.decode(f.properties[seedColTags])
	if err != nil {
		return yacymodel.Seed{}, fmt.Errorf("%w: tags: %w", yacymodel.ErrBadSeed, err)
	}
	seed.Tags = tags

	if raw, ok := f.properties[seedColNews]; ok && raw != "" {
		news, err := peerNewsWireCodec{}.decode(ctx, raw)
		if err != nil {
			return yacymodel.Seed{}, fmt.Errorf("%w: news: %w", yacymodel.ErrBadSeed, err)
		}
		seed.News = yacymodel.Some(news)
	}

	if seedListAddress, ok := f.properties[seedColSeedListAddress]; ok && seedListAddress != "" {
		parsed, err := yacymodel.ParseSeedListURL(seedListAddress)
		if err != nil {
			return yacymodel.Seed{}, fmt.Errorf("%w: seed list url: %w", yacymodel.ErrBadSeed, err)
		}
		seed.SeedListAddress = yacymodel.Some(parsed)
	}

	return seed, nil
}

func (f seedWireForm) host(column string) yacymodel.Optional[yacymodel.Host] {
	value := f.properties[column]
	if value == "" {
		return yacymodel.None[yacymodel.Host]()
	}
	host, err := yacymodel.ParseHost(value)
	if err != nil {
		return yacymodel.None[yacymodel.Host]()
	}
	return yacymodel.Some(host)
}

func (f seedWireForm) additionalAddresses() yacymodel.Optional[[]yacymodel.Host] {
	value := f.properties[seedColAdditionalAddresses]
	if value == "" {
		return yacymodel.None[[]yacymodel.Host]()
	}
	var hosts []yacymodel.Host
	for _, field := range strings.Split(value, seedAddressSeparator) {
		if field == "" {
			continue
		}
		host, err := yacymodel.ParseHost(field)
		if err != nil {
			continue
		}
		hosts = append(hosts, host)
	}
	if len(hosts) == 0 {
		return yacymodel.None[[]yacymodel.Host]()
	}
	return yacymodel.Some(hosts)
}

func (f seedWireForm) port(column string) yacymodel.Optional[yacymodel.Port] {
	value := f.properties[column]
	if value == "" {
		return yacymodel.None[yacymodel.Port]()
	}
	port, err := yacymodel.ParsePort(value)
	if err != nil {
		return yacymodel.None[yacymodel.Port]()
	}
	return yacymodel.Some(port)
}

func (f seedWireForm) peerType(column string) yacymodel.Optional[yacymodel.PeerType] {
	value := f.properties[column]
	if value == "" {
		return yacymodel.None[yacymodel.PeerType]()
	}
	peerType, err := yacymodel.ParsePeerType(value)
	if err != nil {
		return yacymodel.None[yacymodel.PeerType]()
	}
	return yacymodel.Some(peerType)
}

func (f seedWireForm) version() yacymodel.Optional[yacymodel.SoftwareVersion] {
	value := f.properties[seedColVersion]
	if value == "" {
		return yacymodel.None[yacymodel.SoftwareVersion]()
	}
	version, err := softwareVersionWireCodec{}.decode(value)
	if err != nil {
		return yacymodel.None[yacymodel.SoftwareVersion]()
	}
	return yacymodel.Some(version)
}

func (f seedWireForm) solrAvailable() yacymodel.Optional[bool] {
	switch f.properties[seedColSolrAvailable] {
	case seedSolrAvailableYes:
		return yacymodel.Some(true)
	case seedSolrAvailableNo:
		return yacymodel.Some(false)
	default:
		return yacymodel.None[bool]()
	}
}

func (f seedWireForm) timestamp(column string) yacymodel.Optional[time.Time] {
	instant, ok := instantWireCodec{}.decode(f.properties[column])
	if !ok {
		return yacymodel.None[time.Time]()
	}
	return yacymodel.Some(instant)
}

func (f seedWireForm) epochMillis(column string) yacymodel.Optional[time.Time] {
	value := f.properties[column]
	if value == "" {
		return yacymodel.None[time.Time]()
	}
	millis, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return yacymodel.None[time.Time]()
	}
	return yacymodel.Some(time.UnixMilli(millis).UTC())
}

func (f seedWireForm) utcOffset() yacymodel.Optional[yacymodel.UTCOffset] {
	value := f.properties[seedColUTCOffset]
	if value == "" {
		return yacymodel.None[yacymodel.UTCOffset]()
	}
	offset, err := utcOffsetWireCodec{}.decode(value)
	if err != nil {
		return yacymodel.None[yacymodel.UTCOffset]()
	}
	return yacymodel.Some(offset)
}

func (f seedWireForm) integer(column string) int {
	number, err := strconv.Atoi(strings.TrimSpace(f.properties[column]))
	if err != nil {
		return 0
	}
	return number
}

func (f seedWireForm) decimal(column string) float64 {
	number, err := strconv.ParseFloat(strings.TrimSpace(f.properties[column]), 64)
	if err != nil {
		return 0
	}
	return number
}

func seedWireFormFromDomain(seed yacymodel.Seed) seedWireForm {
	f := seedWireForm{properties: map[string]string{}}

	f.put(seedColHash, seed.Hash.String())
	f.put(seedColName, seed.Name.String())
	f.put(seedColPeerType, seed.PeerType.String())

	if host, ok := seed.PrimaryAddress.Get(); ok {
		f.put(seedColPrimaryAddress, host.String())
	}
	if hosts, ok := seed.AdditionalAddresses.Get(); ok {
		names := make([]string, 0, len(hosts))
		for _, host := range hosts {
			names = append(names, host.String())
		}
		f.put(seedColAdditionalAddresses, strings.Join(names, seedAddressSeparator))
	}
	if port, ok := seed.Port.Get(); ok {
		f.put(seedColPort, port.String())
	}
	if port, ok := seed.SecurePort.Get(); ok {
		f.put(seedColSecurePort, port.String())
	}
	if address, ok := seed.SeedListAddress.Get(); ok {
		f.put(seedColSeedListAddress, address.String())
	}
	if peerType, ok := seed.RemotePeerType.Get(); ok {
		f.put(seedColRemotePeerType, peerType.String())
	}
	if capabilities, ok := seed.Capabilities.Get(); ok {
		f.put(seedColCapabilities, peerCapabilitiesWireCodec{}.encode(capabilities))
	}
	if version, ok := seed.Version.Get(); ok {
		f.put(seedColVersion, softwareVersionWireCodec{}.encode(version))
	}
	f.put(seedColTags, peerTagsWireCodec{}.encode(seed.Tags))
	if available, ok := seed.SolrAvailable.Get(); ok {
		f.putSolrAvailable(available)
	}
	if instant, ok := seed.FirstSeen.Get(); ok {
		f.put(seedColFirstSeen, instantWireCodec{}.encode(instant))
	}
	if instant, ok := seed.LastSeen.Get(); ok {
		f.put(seedColLastSeen, instantWireCodec{}.encode(instant))
	}
	if instant, ok := seed.DisconnectedAt.Get(); ok {
		f.put(seedColDisconnectedAt, strconv.FormatInt(instant.UnixMilli(), 10))
	}
	if offset, ok := seed.UTCOffset.Get(); ok {
		f.put(seedColUTCOffset, utcOffsetWireCodec{}.encode(offset))
	}
	f.putActivity(seed)
	if news, ok := seed.News.Get(); ok {
		f.put(seedColNews, peerNewsWireCodec{}.encode(news))
	}

	return f
}

func (f *seedWireForm) putActivity(seed yacymodel.Seed) {
	f.put(seedColUptime, strconv.Itoa(int(seed.Uptime/time.Minute)))
	f.put(seedColIndexingSpeed, strconv.Itoa(seed.IndexingSpeed))
	f.put(seedColRetrievalSpeed, strconv.Itoa(seed.RetrievalSpeed))
	f.put(seedColUplinkSpeed, strconv.Itoa(seed.UplinkSpeed))
	f.put(seedColClientConnectRate, strconv.FormatFloat(seed.ClientConnectRate, 'f', -1, 64))
	f.put(seedColIndexedWords, strconv.Itoa(seed.IndexedWords))
	f.put(seedColStoredURLs, strconv.Itoa(seed.StoredURLs))
	f.put(seedColNoticedURLs, strconv.Itoa(seed.NoticedURLs))
	f.put(seedColRemoteCrawlURLs, strconv.Itoa(seed.RemoteCrawlURLs))
	f.put(seedColStoredSeeds, strconv.Itoa(seed.StoredSeeds))
	f.put(seedColWordsSent, strconv.Itoa(seed.WordsSent))
	f.put(seedColWordsReceived, strconv.Itoa(seed.WordsReceived))
	f.put(seedColURLsSent, strconv.Itoa(seed.URLsSent))
	f.put(seedColURLsReceived, strconv.Itoa(seed.URLsReceived))
}

func (f *seedWireForm) put(column, value string) {
	f.properties[column] = value
	f.columns = append(f.columns, column)
}

func (f *seedWireForm) putSolrAvailable(available bool) {
	if available {
		f.put(seedColSolrAvailable, seedSolrAvailableYes)
		return
	}
	f.put(seedColSolrAvailable, seedSolrAvailableNo)
}

func (f seedWireForm) row() string {
	var b strings.Builder
	b.WriteByte(propertyOpen)
	for i, column := range f.columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(column)
		b.WriteByte('=')
		b.WriteString(f.properties[column])
	}
	b.WriteByte(propertyClose)
	return b.String()
}

func (f seedWireForm) framed() string {
	return encodeBase64WireForm(f.row())
}
