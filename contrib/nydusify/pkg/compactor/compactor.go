package compactor

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/dragonflyoss/nydus/contrib/nydusify/pkg/build"
	"github.com/pkg/errors"
)

var defaultCompactConfig = &CompactConfig{
	MinUsedRatio:    "5",
	CompactBlobSize: "10485760",
	MaxCompactSize:  "104857600",
	LayersToCompact: "32",
}

type CompactConfig struct {
	MinUsedRatio    string
	CompactBlobSize string
	MaxCompactSize  string
	LayersToCompact string
	BlobsDir        string
}

func (cfg *CompactConfig) Dumps(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()
	if err = json.NewEncoder(file).Encode(cfg); err != nil {
		return errors.Wrap(err, "failed to encode json")
	}
	return nil
}

func loadCompactConfig(filePath string) (CompactConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return CompactConfig{}, errors.Wrap(err, "failed to load compact configuration file")
	}
	defer file.Close()
	var cfg CompactConfig
	if err = json.NewDecoder(file).Decode(&cfg); err != nil {
		return CompactConfig{}, errors.Wrap(err, "failed to decode compact configuration file")
	}
	return cfg, nil
}

type Compactor struct {
	builder *build.Builder
	workdir string
	cfg     CompactConfig
}

func NewCompactor(nydusImagePath, workdir, configPath string) (*Compactor, error) {
	var (
		cfg CompactConfig
		err error
	)
	if configPath != "" {
		cfg, err = loadCompactConfig(configPath)
		if err != nil {
			return nil, errors.Wrap(err, "compact config err")
		}
	} else {
		cfg = *defaultCompactConfig
	}
	cfg.BlobsDir = workdir
	return &Compactor{
		builder: build.NewBuilder(nydusImagePath),
		workdir: workdir,
		cfg:     cfg,
	}, nil
}

func (compactor *Compactor) Compact(bootstrapPath, chunkDict, backendType, backendConfigFile string) (string, error) {
	targetBootstrap := bootstrapPath + ".compact"
	if err := os.Remove(targetBootstrap); err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, "failed to delete old bootstrap file")
	}
	outputJSONPath := filepath.Join(compactor.workdir, "compact-result.json")
	if err := os.Remove(outputJSONPath); err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, "failed to delete old output-json file")
	}
	err := compactor.builder.Compact(build.CompactOption{
		ChunkDict:           chunkDict,
		BootstrapPath:       bootstrapPath,
		OutputBootstrapPath: targetBootstrap,
		BackendType:         backendType,
		BackendConfigPath:   backendConfigFile,
		OutputJSONPath:      outputJSONPath,
		MinUsedRatio:        compactor.cfg.MinUsedRatio,
		CompactBlobSize:     compactor.cfg.CompactBlobSize,
		MaxCompactSize:      compactor.cfg.MaxCompactSize,
		LayersToCompact:     compactor.cfg.LayersToCompact,
		BlobsDir:            compactor.cfg.BlobsDir,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to run compact command")
	}

	return targetBootstrap, nil
}
