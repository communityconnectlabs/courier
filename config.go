package courier

import "github.com/nyaruka/ezconf"

// Config is our top level configuration object
type Config struct {
	Backend                    string `help:"the backend that will be used by courier (currently only rapidpro is supported)"`
	SentryDSN                  string `help:"the DSN used for logging errors to Sentry"`
	Domain                     string `help:"the domain courier is exposed on"`
	Address                    string `help:"the network interface address courier will bind to"`
	Port                       int    `help:"the port courier will listen on"`
	DB                         string `help:"URL describing how to connect to the RapidPro database"`
	Redis                      string `help:"URL describing how to connect to Redis"`
	SpoolDir                   string `help:"the local directory where courier will write statuses or msgs that need to be retried (needs to be writable)"`
	S3PublicAccessEndpoint     string `help:"the public endpoint that can be used to get the private files"`
	S3Endpoint                 string `help:"the S3 endpoint we will write attachments to"`
	S3Region                   string `help:"the S3 region we will write attachments to"`
	S3MediaBucket              string `help:"the S3 bucket we will write attachments to"`
	S3MediaPrefix              string `help:"the prefix that will be added to attachment filenames"`
	S3DisableSSL               bool   `help:"whether we disable SSL when accessing S3. Should always be set to False unless you're hosting an S3 compatible service within a secure internal network"`
	S3ForcePathStyle           bool   `help:"whether we force S3 path style. Should generally need to default to False unless you're hosting an S3 compatible service"`
	AWSAccessKeyID             string `help:"the access key id to use when authenticating S3"`
	AWSSecretAccessKey         string `help:"the secret access key id to use when authenticating S3"`
	FacebookApplicationSecret  string `help:"the Facebook app secret"`
	FacebookWebhookSecret      string `help:"the secret for Facebook webhook URL verification"`
	MaxWorkers                 int    `help:"the maximum number of go routines that will be used for sending (set to 0 to disable sending)"`
	LibratoUsername            string `help:"the username that will be used to authenticate to Librato"`
	LibratoToken               string `help:"the token that will be used to authenticate to Librato"`
	StatusUsername             string `help:"the username that is needed to authenticate against the /status endpoint"`
	StatusPassword             string `help:"the password that is needed to authenticate against the /status endpoint"`
	LogLevel                   string `help:"the logging level courier should use"`
	Version                    string `help:"the version that will be used in request and response headers"`
	WebChatServerSecret        string `help:"key for encoding the websocket tokens"`
	SMPPServerEndpoint         string `help:"the URL of the server that handles SMPP connections"`
	SMPPServerToken            string `help:"the token of the server that handles SMPP connections"`
	KaleyraMMSLongcodeEndpoint string `help:"the Kaleyra endpoint for long code MMS service"`
	KaleyraMMSEndpoint         string `help:"the Kaleyra endpoint for MMS service"`
	KaleyraMMSUsername         string `help:"the Kaleyra username for MMS service authentication"`
	KaleyraMMSPassword         string `help:"the Kaleyra password for MMS service authentication"`

	// IncludeChannels is the list of channels to enable, empty means include all
	IncludeChannels []string

	// ExcludeChannels is the list of channels to exclude, empty means exclude none
	ExcludeChannels []string
}

// NewConfig returns a new default configuration object
func NewConfig() *Config {
	return &Config{
		Backend:                   "rapidpro",
		Domain:                    "localhost",
		Address:                   "",
		Port:                      8080,
		DB:                        "postgres://temba:temba@localhost/temba?sslmode=disable",
		Redis:                     "redis://localhost:6379/15",
		SpoolDir:                  "/var/spool/courier",
		S3PublicAccessEndpoint:    "http://localhost:8000/storage",
		S3Endpoint:                "https://s3.amazonaws.com",
		S3Region:                  "us-east-1",
		S3MediaBucket:             "courier-media",
		S3MediaPrefix:             "/media/",
		S3DisableSSL:              false,
		S3ForcePathStyle:          false,
		AWSAccessKeyID:            "",
		AWSSecretAccessKey:        "",
		FacebookApplicationSecret: "missing_facebook_app_secret",
		FacebookWebhookSecret:     "missing_facebook_webhook_secret",
		MaxWorkers:                32,
		LogLevel:                  "error",
		Version:                   "Dev",
		WebChatServerSecret:       "",
		SMPPServerEndpoint:        "",
		SMPPServerToken:           "",
	}
}

// LoadConfig loads our configuration from the passed in filename
func LoadConfig(filename string) *Config {
	config := NewConfig()
	loader := ezconf.NewLoader(
		config,
		"courier", "Courier - A fast message broker for SMS and IP messages",
		[]string{filename},
	)

	loader.MustLoad()
	return config
}
