/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package externalbuilder

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/osdi23p228/fabric/common/flogging"
	"github.com/osdi23p228/fabric/core/container/ccintf"
	"github.com/osdi23p228/fabric/core/peer"
	"github.com/pkg/errors"
)

var (
	// DefaultPropagateEnvironment enumerates the list of environment variables that are
	// implicitly propagated to external builder and launcher commands.
	DefaultPropagateEnvironment = []string{"LD_LIBRARY_PATH", "LIBPATH", "PATH", "TMPDIR"}

	logger = flogging.MustGetLogger("chaincode.externalbuilder")
)

// BuildInfo contains metadata is that is saved to the local file system with the
// assets generated by an external builder. This is used to associate build output
// with the builder that generated it.
type BuildInfo struct {
	// BuilderName is the user provided name of the external builder.
	BuilderName string `json:"builder_name"`
}

// A Detector is responsible for orchestrating the external builder detection and
// build process.
type Detector struct {
	// DurablePath is the file system location where chaincode assets are persisted.
	DurablePath string
	// Builders are the builders that detect and build processing will use.
	Builders []*Builder
}

// CachedBuild returns a build instance that was already built or nil when no
// instance has been found.  An error is returned only when an unexpected
// condition is encountered.
func (d *Detector) CachedBuild(ccid string) (*Instance, error) {
	durablePath := filepath.Join(d.DurablePath, SanitizeCCIDPath(ccid))
	_, err := os.Stat(durablePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithMessage(err, "existing build detected, but something went wrong inspecting it")
	}

	buildInfoPath := filepath.Join(durablePath, "build-info.json")
	buildInfoData, err := ioutil.ReadFile(buildInfoPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "could not read '%s' for build info", buildInfoPath)
	}

	var buildInfo BuildInfo
	if err := json.Unmarshal(buildInfoData, &buildInfo); err != nil {
		return nil, errors.WithMessagef(err, "malformed build info at '%s'", buildInfoPath)
	}

	for _, builder := range d.Builders {
		if builder.Name == buildInfo.BuilderName {
			return &Instance{
				PackageID:   ccid,
				Builder:     builder,
				BldDir:      filepath.Join(durablePath, "bld"),
				ReleaseDir:  filepath.Join(durablePath, "release"),
				TermTimeout: 5 * time.Second,
			}, nil
		}
	}

	return nil, errors.Errorf("chaincode '%s' was already built with builder '%s', but that builder is no longer available", ccid, buildInfo.BuilderName)
}

// Build executes the external builder detect and build process.
//
// Before running the detect and build process, the detector first checks the
// durable path for the results of a previous build for the provided package.
// If found, the detect and build process is skipped and the existing instance
// is returned.
func (d *Detector) Build(ccid string, mdBytes []byte, codeStream io.Reader) (*Instance, error) {
	// A small optimization: prevent exploding the build package out into the
	// file system unless there are external builders defined.
	if len(d.Builders) == 0 {
		return nil, nil
	}

	// Look for a cached instance.
	i, err := d.CachedBuild(ccid)
	if err != nil {
		return nil, errors.WithMessage(err, "existing build could not be restored")
	}
	if i != nil {
		return i, nil
	}

	buildContext, err := NewBuildContext(ccid, mdBytes, codeStream)
	if err != nil {
		return nil, errors.WithMessage(err, "could not create build context")
	}
	defer buildContext.Cleanup()

	builder := d.detect(buildContext)
	if builder == nil {
		logger.Debugf("no external builder detected for %s", ccid)
		return nil, nil
	}

	if err := builder.Build(buildContext); err != nil {
		return nil, errors.WithMessage(err, "external builder failed to build")
	}

	if err := builder.Release(buildContext); err != nil {
		return nil, errors.WithMessage(err, "external builder failed to release")
	}

	durablePath := filepath.Join(d.DurablePath, SanitizeCCIDPath(ccid))

	err = os.Mkdir(durablePath, 0700)
	if err != nil {
		return nil, errors.WithMessagef(err, "could not create dir '%s' to persist build output", durablePath)
	}

	buildInfo, err := json.Marshal(&BuildInfo{
		BuilderName: builder.Name,
	})
	if err != nil {
		os.RemoveAll(durablePath)
		return nil, errors.WithMessage(err, "could not marshal for build-info.json")
	}

	err = ioutil.WriteFile(filepath.Join(durablePath, "build-info.json"), buildInfo, 0600)
	if err != nil {
		os.RemoveAll(durablePath)
		return nil, errors.WithMessage(err, "could not write build-info.json")
	}

	durableReleaseDir := filepath.Join(durablePath, "release")
	err = CopyDir(logger, buildContext.ReleaseDir, durableReleaseDir)
	if err != nil {
		return nil, errors.WithMessagef(err, "could not move or copy build context release to persistent location '%s'", durablePath)
	}

	durableBldDir := filepath.Join(durablePath, "bld")
	err = CopyDir(logger, buildContext.BldDir, durableBldDir)
	if err != nil {
		return nil, errors.WithMessagef(err, "could not move or copy build context bld to persistent location '%s'", durablePath)
	}

	return &Instance{
		PackageID:   ccid,
		Builder:     builder,
		BldDir:      durableBldDir,
		ReleaseDir:  durableReleaseDir,
		TermTimeout: 5 * time.Second,
	}, nil
}

func (d *Detector) detect(buildContext *BuildContext) *Builder {
	for _, builder := range d.Builders {
		if builder.Detect(buildContext) {
			return builder
		}
	}
	return nil
}

// BuildContext holds references to the various assets locations necessary to
// execute the detect, build, release, and run programs for external builders
type BuildContext struct {
	CCID        string
	ScratchDir  string
	SourceDir   string
	ReleaseDir  string
	MetadataDir string
	BldDir      string
}

// NewBuildContext creates the directories required to runt he external
// build process and extracts the chaincode package assets.
//
// Users of the BuildContext must call Cleanup when the build process is
// complete to remove the transient file system assets.
func NewBuildContext(ccid string, mdBytes []byte, codePackage io.Reader) (bc *BuildContext, err error) {
	scratchDir, err := ioutil.TempDir("", "fabric-"+SanitizeCCIDPath(ccid))
	if err != nil {
		return nil, errors.WithMessage(err, "could not create temp dir")
	}

	defer func() {
		if err != nil {
			os.RemoveAll(scratchDir)
		}
	}()

	sourceDir := filepath.Join(scratchDir, "src")
	if err = os.Mkdir(sourceDir, 0700); err != nil {
		return nil, errors.WithMessage(err, "could not create source dir")
	}

	metadataDir := filepath.Join(scratchDir, "metadata")
	if err = os.Mkdir(metadataDir, 0700); err != nil {
		return nil, errors.WithMessage(err, "could not create metadata dir")
	}

	outputDir := filepath.Join(scratchDir, "bld")
	if err = os.Mkdir(outputDir, 0700); err != nil {
		return nil, errors.WithMessage(err, "could not create build dir")
	}

	releaseDir := filepath.Join(scratchDir, "release")
	if err = os.Mkdir(releaseDir, 0700); err != nil {
		return nil, errors.WithMessage(err, "could not create release dir")
	}

	err = Untar(codePackage, sourceDir)
	if err != nil {
		return nil, errors.WithMessage(err, "could not untar source package")
	}

	err = ioutil.WriteFile(filepath.Join(metadataDir, "metadata.json"), mdBytes, 0700)
	if err != nil {
		return nil, errors.WithMessage(err, "could not write metadata file")
	}

	return &BuildContext{
		ScratchDir:  scratchDir,
		SourceDir:   sourceDir,
		MetadataDir: metadataDir,
		BldDir:      outputDir,
		ReleaseDir:  releaseDir,
		CCID:        ccid,
	}, nil
}

// Cleanup removes the build context artifacts.
func (bc *BuildContext) Cleanup() {
	os.RemoveAll(bc.ScratchDir)
}

var pkgIDreg = regexp.MustCompile("[<>:\"/\\\\|\\?\\*&]")

// SanitizeCCIDPath is used to ensure that special characters are removed from
// file names.
func SanitizeCCIDPath(ccid string) string {
	return pkgIDreg.ReplaceAllString(ccid, "-")
}

// A Builder is used to interact with an external chaincode builder and launcher.
type Builder struct {
	PropagateEnvironment []string
	Location             string
	Logger               *flogging.FabricLogger
	Name                 string
	MSPID                string
}

// CreateBuilders will construct builders from the peer configuration.
func CreateBuilders(builderConfs []peer.ExternalBuilder, mspid string) []*Builder {
	var builders []*Builder
	for _, builderConf := range builderConfs {
		builders = append(builders, &Builder{
			Location:             builderConf.Path,
			Name:                 builderConf.Name,
			PropagateEnvironment: builderConf.PropagateEnvironment,
			Logger:               logger.Named(builderConf.Name),
			MSPID:                mspid,
		})
	}
	return builders
}

// Detect runs the `detect` script.
func (b *Builder) Detect(buildContext *BuildContext) bool {
	detect := filepath.Join(b.Location, "bin", "detect")
	cmd := b.NewCommand(detect, buildContext.SourceDir, buildContext.MetadataDir)

	err := b.runCommand(cmd)
	if err != nil {
		logger.Debugf("builder '%s' detect failed: %s", b.Name, err)
		return false
	}

	return true
}

// Build runs the `build` script.
func (b *Builder) Build(buildContext *BuildContext) error {
	build := filepath.Join(b.Location, "bin", "build")
	cmd := b.NewCommand(build, buildContext.SourceDir, buildContext.MetadataDir, buildContext.BldDir)

	err := b.runCommand(cmd)
	if err != nil {
		return errors.Wrapf(err, "external builder '%s' failed", b.Name)
	}

	return nil
}

// Release runs the `release` script.
func (b *Builder) Release(buildContext *BuildContext) error {
	release := filepath.Join(b.Location, "bin", "release")

	_, err := exec.LookPath(release)
	if err != nil {
		b.Logger.Debugf("Skipping release step for '%s' as no release binary found", buildContext.CCID)
		return nil
	}

	cmd := b.NewCommand(release, buildContext.BldDir, buildContext.ReleaseDir)
	err = b.runCommand(cmd)
	if err != nil {
		return errors.Wrapf(err, "builder '%s' release failed", b.Name)
	}

	return nil
}

// runConfig is serialized to disk when launching.
type runConfig struct {
	CCID        string `json:"chaincode_id"`
	PeerAddress string `json:"peer_address"`
	ClientCert  string `json:"client_cert"` // PEM encoded client certificate
	ClientKey   string `json:"client_key"`  // PEM encoded client key
	RootCert    string `json:"root_cert"`   // PEM encoded peer chaincode certificate
	MSPID       string `json:"mspid"`
}

func newRunConfig(ccid string, peerConnection *ccintf.PeerConnection, mspid string) runConfig {
	var tlsConfig ccintf.TLSConfig
	if peerConnection.TLSConfig != nil {
		tlsConfig = *peerConnection.TLSConfig
	}

	return runConfig{
		PeerAddress: peerConnection.Address,
		CCID:        ccid,
		ClientCert:  string(tlsConfig.ClientCert),
		ClientKey:   string(tlsConfig.ClientKey),
		RootCert:    string(tlsConfig.RootCert),
		MSPID:       mspid,
	}
}

// Run starts the `run` script and returns a Session that can be used to
// signal it and wait for termination.
func (b *Builder) Run(ccid, bldDir string, peerConnection *ccintf.PeerConnection) (*Session, error) {
	launchDir, err := ioutil.TempDir("", "fabric-run")
	if err != nil {
		return nil, errors.WithMessage(err, "could not create temp run dir")
	}

	rc := newRunConfig(ccid, peerConnection, b.MSPID)
	marshaledRC, err := json.Marshal(rc)
	if err != nil {
		return nil, errors.WithMessage(err, "could not marshal run config")
	}

	if err := ioutil.WriteFile(filepath.Join(launchDir, "chaincode.json"), marshaledRC, 0600); err != nil {
		return nil, errors.WithMessage(err, "could not write root cert")
	}

	run := filepath.Join(b.Location, "bin", "run")
	cmd := b.NewCommand(run, bldDir, launchDir)
	sess, err := Start(b.Logger, cmd, func(error) { os.RemoveAll(launchDir) })
	if err != nil {
		os.RemoveAll(launchDir)
		return nil, errors.Wrapf(err, "builder '%s' run failed to start", b.Name)
	}

	return sess, nil
}

// runCommand runs a command and waits for it to complete.
func (b *Builder) runCommand(cmd *exec.Cmd) error {
	sess, err := Start(b.Logger, cmd)
	if err != nil {
		return err
	}
	return sess.Wait()
}

// NewCommand creates an exec.Cmd that is configured to prune the calling
// environment down to the environment variables specified in the external
// builder's PropagateEnvironment and the DefaultPropagateEnvironment.
func (b *Builder) NewCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	propagationList := appendDefaultPropagateEnvironment(b.PropagateEnvironment)
	for _, key := range propagationList {
		if val, ok := os.LookupEnv(key); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return cmd
}

func appendDefaultPropagateEnvironment(propagateEnvironment []string) []string {
	for _, variable := range DefaultPropagateEnvironment {
		if !contains(propagateEnvironment, variable) {
			propagateEnvironment = append(propagateEnvironment, variable)
		}
	}
	return propagateEnvironment
}

func contains(propagateEnvironment []string, key string) bool {
	for _, variable := range propagateEnvironment {
		if key == variable {
			return true
		}
	}
	return false
}
