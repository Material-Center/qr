package config

type Extra struct {
	ExtractURL       string `mapstructure:"extract-url" json:"extract-url" yaml:"extract-url"`
	UploadArchiveDir string `mapstructure:"upload-archive-dir" json:"upload-archive-dir" yaml:"upload-archive-dir"`
}
