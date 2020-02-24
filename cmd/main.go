package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cnative/pkg/log"
)

var (
	cfgFile     string
	showVersion bool
	appName     = "example-app"
	version     = "unknown"
	gitCommit   = "unknown"
	srvCfg      serverConfig
)

var rootCmd = &cobra.Command{
	Use:               "example-app",
	Short:             "Example App",
	PersistentPreRunE: initConfig,
	SilenceUsage:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// initConfig reads in config file and ENV variables if set.
func initConfig(cmd *cobra.Command, args []string) error {

	if showVersion {
		printVersion()
		os.Exit(1)
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		// Search config in home directory with name ".reports" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".example")
	}

	viper.SetEnvPrefix("example")
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	applyBaseCLIArgs(&srvCfg)

	if cmd.Name() == "example-app" {
		return nil
	}

	return validateBaseCLIArgs(&srvCfg)
}

func main() {

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.example.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "print the version")

	applyBaseServerConfig(rootCmd) // base server config
	applyTlSFlags(rootCmd)         // TLS Flags
	applyTraceConfig(rootCmd)      // Opencensus agent based tracing
	applyOIDCFlags(rootCmd)        // OIDC Auth Flags
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func applyBaseCLIArgs(sc *serverConfig) {

	// base attributes
	sc.debug = viper.GetBool("debug")
	sc.dPort = uint(viper.GetInt("debug-port"))
	sc.hPort = uint(viper.GetInt("health-port"))
	sc.mPort = uint(viper.GetInt("metrics-port"))
	sc.skipProcessMetrics = viper.GetBool("skip-process-metrics")

	// tls attributes
	sc.tls.certFile = viper.GetString("tls-cert-file")
	sc.tls.skip = viper.GetBool("insecure-skip-tls")
	sc.tls.keyFile = viper.GetString("tls-private-key-file")
	sc.tls.caFile = viper.GetString("client-ca-file")

	//trace attributes
	sc.ocAgent.traceEnabled = viper.GetBool("no-trace")
	sc.ocAgent.host = viper.GetString("oc-agent-host")
	sc.ocAgent.port = uint(viper.GetInt("oc-agent-port"))
	sc.ocAgent.namespace = viper.GetString("oc-namespace")

	// auth attributes
	sc.auth.disabled = viper.GetBool("insecure-no-auth")
	sc.auth.issuerURL = viper.GetString("oidc-issuer-url")
	sc.auth.clientID = viper.GetString("oidc-client-id")
	sc.auth.caFile = viper.GetString("oidc-ca-file")
	sc.auth.requiredClaims = viper.GetStringMapString("oidc-required-claim")
	sc.auth.signingAlgos = viper.GetStringSlice("oidc-signing-algos")
}

func validateBaseCLIArgs(sc *serverConfig) error {

	if !sc.tls.skip && (sc.tls.keyFile == "" || sc.tls.certFile == "") {
		return errors.New("tls cert and key files are not specified. use tls-cert-file & tls-private-key-file")
	}

	if err := sc.tls.resolveAbsFilePath(); err != nil {
		return err
	}

	return nil
}

func getRootLogger(debug bool, serviceName string) (*log.Logger, error) {
	ll := log.InfoLevel
	if debug {
		ll = log.DebugLevel
	}
	name := fmt.Sprintf("%s-%s", appName, serviceName)

	return log.New(log.WithName(name), log.WithLevel(ll))
}

func (t *tlsConfig) resolveAbsFilePath() error {

	if t.certFile != "" {
		f, err := filepath.Abs(t.certFile)
		if err != nil {
			return err
		}
		t.certFile = f
	}

	if t.keyFile != "" {
		f, err := filepath.Abs(t.keyFile)
		if err != nil {
			return err
		}
		t.keyFile = f
	}

	if t.caFile != "" {
		f, err := filepath.Abs(t.caFile)
		if err != nil {
			return err
		}
		t.caFile = f
	}

	return nil
}

// Tries to find out when this binary was compiled.
// Returns the current time if it fails to find it.
func compileTime() time.Time {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}

func printVersion() {
	fmt.Printf("%s\n Version:  %s\n Git Commit:  %s\n Go Version:  %s\n OS/Arch:  %s/%s\n Built:  %s\n",
		appName, version, gitCommit, runtime.Version(), runtime.GOOS, runtime.GOARCH, compileTime())
}
