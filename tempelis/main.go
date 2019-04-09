package main

import "flag"

type options struct {
	DryRun bool
	Config string
}

func parseOptions() options {
	var o options
	flag.BoolVar(&o.DryRun, "dry-run", true, "does nothing if true (which is the default)")
	flag.StringVar(&o.Config, "config", "", "path to a configuration file, or directory of files")
	flag.Parse()
	return o
}

func main() {

}
