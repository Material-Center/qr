package config

type Local struct {
	Path      string `mapstructure:"path" json:"path" yaml:"path"`                   // 本地文件访问路径
	StorePath string `mapstructure:"store-path" json:"store-path" yaml:"store-path"` // 本地文件存储路径
	// 可选：将本地目录挂载为静态资源（自行 rsync/scp 等到该目录）；须与 script-static-url-prefix 同时配置才生效
	ScriptStaticDir       string `mapstructure:"script-static-dir" json:"script-static-dir" yaml:"script-static-dir"`
	ScriptStaticURLPrefix string `mapstructure:"script-static-url-prefix" json:"script-static-url-prefix" yaml:"script-static-url-prefix"`
}
