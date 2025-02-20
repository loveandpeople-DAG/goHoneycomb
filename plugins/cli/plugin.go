package cli

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/tcnksm/go-latest"

	"github.com/loveandpeople-DAG/goHive/daemon"
	"github.com/loveandpeople-DAG/goHive/events"
	"github.com/loveandpeople-DAG/goHive/logger"
	"github.com/loveandpeople-DAG/goHive/node"
	"github.com/loveandpeople-DAG/goHive/timeutil"

	"github.com/loveandpeople-DAG/goBee/pkg/config"
	"github.com/loveandpeople-DAG/goBee/pkg/profile"
	"github.com/loveandpeople-DAG/goBee/pkg/shutdown"
)

var (
	// AppVersion version number
	AppVersion          = "0.5.3"
	LatestGithubVersion = AppVersion

	// AppName app code name
	AppName = "HORNET"

	githubTag *latest.GithubTag
)

var (
	PLUGIN = node.NewPlugin("CLI", node.Enabled, configure, run)
	log    *logger.Logger
)

func onAddPlugin(name string, status int) {
	AddPluginStatus(node.GetPluginIdentifier(name), status)
}

func init() {
	for name, plugin := range node.GetPlugins() {
		onAddPlugin(name, plugin.Status)
	}

	node.Events.AddPlugin.Attach(events.NewClosure(onAddPlugin))

	flag.Usage = printUsage
}

func parseParameters() {
	for _, pluginName := range config.NodeConfig.GetStringSlice(node.CFG_DISABLE_PLUGINS) {
		node.DisabledPlugins[strings.ToLower(pluginName)] = true
	}
	for _, pluginName := range config.NodeConfig.GetStringSlice(node.CFG_ENABLE_PLUGINS) {
		node.EnabledPlugins[strings.ToLower(pluginName)] = true
	}
}

func configure(plugin *node.Plugin) {

	log = logger.NewLogger(plugin.Name)

	githubTag = &latest.GithubTag{
		Owner:             "gohornet",
		Repository:        "hornet",
		FixVersionStrFunc: fixVersion,
		TagFilterFunc:     includeVersionInCheck,
	}

	fmt.Printf(`
              ██╗  ██╗ ██████╗ ██████╗ ███╗   ██╗███████╗████████╗
              ██║  ██║██╔═══██╗██╔══██╗████╗  ██║██╔════╝╚══██╔══╝
              ███████║██║   ██║██████╔╝██╔██╗ ██║█████╗     ██║
              ██╔══██║██║   ██║██╔══██╗██║╚██╗██║██╔══╝     ██║
              ██║  ██║╚██████╔╝██║  ██║██║ ╚████║███████╗   ██║
              ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝
                                   v%s
`+"\n\n", AppVersion)

	checkLatestVersion()

	if config.NodeConfig.GetString(config.CfgProfileUseProfile) == config.AutoProfileName {
		log.Infof("Profile mode 'auto', Using profile '%s'", profile.LoadProfile().Name)
	} else {
		log.Infof("Using profile '%s'", profile.LoadProfile().Name)
	}

	log.Info("Loading plugins ...")
}

func fixVersion(version string) string {
	ver := strings.Replace(version, "v", "", 1)
	if !strings.Contains(ver, "-rc.") {
		ver = strings.Replace(ver, "-rc", "-rc.", 1)
	}
	return ver
}

func includeVersionInCheck(version string) bool {
	isPrerelease := func(ver string) bool {
		return strings.Contains(ver, "-rc")
	}

	if isPrerelease(AppVersion) {
		// When using pre-release versions, check for any updates
		return true
	}

	return !isPrerelease(version)
}

func checkLatestVersion() {

	res, err := latest.Check(githubTag, fixVersion(AppVersion))
	if err != nil {
		log.Warnf("Update check failed: %s", err.Error())
		return
	}

	if res.Outdated {
		log.Infof("Update to %s available on https://github.com/loveandpeople-DAG/goBee/releases/latest", res.Current)
		LatestGithubVersion = res.Current
	}
}

func run(_ *node.Plugin) {

	// create a background worker that checks for latest version every hour
	daemon.BackgroundWorker("Version update checker", func(shutdownSignal <-chan struct{}) {
		timeutil.Ticker(checkLatestVersion, 1*time.Hour, shutdownSignal)
	}, shutdown.PriorityUpdateCheck)
}
