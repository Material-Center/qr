package config

type Extra struct {
	ExtractURL       string `mapstructure:"extract-url" json:"extract-url" yaml:"extract-url"`
	UploadArchiveDir string `mapstructure:"upload-archive-dir" json:"upload-archive-dir" yaml:"upload-archive-dir"`
	MIEnvInternalKey string `mapstructure:"mi-env-internal-key" json:"mi-env-internal-key" yaml:"mi-env-internal-key"`
}
