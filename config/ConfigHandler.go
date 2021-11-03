package config

import (
	"flag"
	"fmt"
	"os"

	logging "github.com/op/go-logging"
	yaml "gopkg.in/yaml.v3"
)

var (
	log        = logging.MustGetLogger("HoneyBEE")
	GConfig    *Config
	MemProfile string
	CpuProfile string
)

// Config struct for HoneyBEE config
type Config struct {
	Proxy struct {
		Host     string `yaml:"host"`    //IP Address to bind the Server
		DEBUG    bool   `yaml:"debug"`   //Output DEBUG info -- TBD
		Timeout  int    `yaml:"timeout"` // Server timeout to use until a connection is destroyed when unresponsive (in seconds)
		Protocol struct {
			AvailableProtocols []int `yaml:"available-protocols"`
		} `yaml:"protocol"`
	} `yaml:"proxy-server"`
	Backends struct {
		Servers              []string `yaml:"servers"`
		LoadBalancingServers []string `yaml:"load-balancing-servers"`
	} `yaml:"backends"`
	Performance struct {
		CPU                       int    `yaml:"cpu"`
		GCPercent                 int    `yaml:"gc-percent"`
		PacketsPerSecond          uint32 `yaml:"packets-per-second"` //TBD
		ApplyStrictMovementChecks bool   `yaml:"movement-checks"`    //TBD
		LimboMode                 bool   `yaml:"limbo-mode-when-backend-down"`
		CheckServerSeconds        int    `yaml:"check-server-seconds"`
		Listeners                 int    `yaml:"listeners"`
	} `yaml:"performance"`
}

// NewConfig returns a new decoded Config struct
func NewConfig(configPath string) (*Config, error) {
	config := new(Config)
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	d := yaml.NewDecoder(file) //Create new YAML decode
	//Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}
	file.Close()
	return config, nil
}

//ValidateConfigPath - makes sure that the path provided is a file that can be read
func ValidateConfigPath(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

var configPath string

//ParseFlags - will create and parse the CLI flags and return the path to be used
func ParseFlags() (string, error) {
	//var configPath string
	//Set up a CLI flag "-config" to allow users to supply the configuration file - defaults to config.yml
	flag.StringVar(&configPath, "config", "./config.yml", "path to config file")
	flag.StringVar(&MemProfile, "memprofile", "", "write memory profile to this file")
	flag.StringVar(&CpuProfile, "cpuprofile", "", "write cpu profile to file")
	//Parse the flags
	flag.Parse()
	//Validate the path
	if err := ValidateConfigPath(configPath); err != nil {
		return "", err
	}
	//Return the configuration path
	return configPath, nil
}

//ConfigStart - Handles the config struct creation
func ConfigStart() *Config {
	//Create config struct
	cfgPath, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := NewConfig(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	GConfig = cfg
	return cfg
}

func GetConfig() *Config {
	return GConfig
}

func ConfigReload() {
	var err error
	GConfig, err = NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
}
