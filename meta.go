package main

// Version - build version of massivedl.Version
// Should be specified during build: -ldflags "-X main.Version=1.0.1"
var Version = "No version provided during build"

// Buildstamp - the timestamp of the build time
// Should be specified during build: -ldflags "-X main.Buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'`"
var Buildstamp = "No buildstamp provided during build"

// Githash - hash of head commit during build
// Should be specified during build: -ldflags "-X main.Githash=`git rev-parse HEAD`"
var Githash = "No githash provided during build"
