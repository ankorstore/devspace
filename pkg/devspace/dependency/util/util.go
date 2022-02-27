package util

import (
	"fmt"
	"github.com/loft-sh/devspace/pkg/util/encoding"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/util/git"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

var authRegEx = regexp.MustCompile(`^(https?:\/\/)[^:]+:[^@]+@(.*)$`)

// DependencyFolder is the dependency folder in the home directory of the user
const DependencyFolder = ".devspace/dependencies"

// DependencyFolderPath will be filled during init
var DependencyFolderPath string

func init() {
	// Make sure dependency folder exists locally
	homedir, _ := homedir.Dir()

	DependencyFolderPath = filepath.Join(homedir, filepath.FromSlash(DependencyFolder))
}

func DownloadDependency(basePath string, source *latest.SourceConfig, log log.Logger) (configPath string, err error) {
	ID, err := getDependencyID(source)
	if err != nil {
		return "", err
	}

	// Resolve source
	var localPath string
	if source.Git != "" {
		gitPath := strings.TrimSpace(source.Git)

		_ = os.MkdirAll(DependencyFolderPath, 0755)
		localPath = filepath.Join(DependencyFolderPath, encoding.Convert(ID))

		// Check if dependency exists
		_, statErr := os.Stat(localPath)

		// Update dependency
		if !source.DisablePull || statErr != nil {
			repo, err := git.NewGitCLIRepository(localPath)
			if err != nil {
				if statErr == nil {
					log.Warnf("Error creating git cli: %v", err)
					return getDependencyConfigPath(localPath, source)
				}

				return "", err
			}

			err = repo.Clone(git.CloneOptions{
				URL:            gitPath,
				Tag:            source.Tag,
				Branch:         source.Branch,
				Commit:         source.Revision,
				Args:           source.CloneArgs,
				DisableShallow: source.DisableShallow,
			})
			if err != nil {
				if statErr == nil {
					log.Warnf("Error cloning or pulling git repository %s: %v", gitPath, err)
					return getDependencyConfigPath(localPath, source)
				}

				return "", errors.Wrap(err, "clone repository")
			}

			log.Debugf("Pulled %s", gitPath)
		}
	} else if source.Path != "" {
		if isURL(source.Path) {
			localPath = filepath.Join(DependencyFolderPath, encoding.Convert(ID))
			_ = os.MkdirAll(localPath, 0755)

			// Check if dependency exists
			configPath := filepath.Join(localPath, constants.DefaultConfigPath)
			_, statErr := os.Stat(configPath)

			if !source.DisablePull || statErr != nil {
				// Create the file
				out, err := os.Create(configPath)
				if err != nil {
					if statErr == nil {
						log.Warnf("Error creating file: %v", err)
						return getDependencyConfigPath(localPath, source)
					}

					return "", err
				}
				defer out.Close()

				// Get the data
				resp, err := http.Get(source.Path)
				if err != nil {
					if statErr == nil {
						log.Warnf("Error retrieving url %s: %v", source.Path, err)
						return getDependencyConfigPath(localPath, source)
					}

					return "", errors.Wrapf(err, "request %s", source.Path)
				}
				defer resp.Body.Close()

				// Write the body to file
				_, err = io.Copy(out, resp.Body)
				if err != nil {
					if statErr == nil {
						log.Warnf("Error retrieving url %s: %v", source.Path, err)
						return getDependencyConfigPath(localPath, source)
					}

					return "", errors.Wrapf(err, "download %s", source.Path)
				}
			}
		} else {
			if filepath.IsAbs(source.Path) {
				localPath = source.Path
			} else {
				localPath, err = filepath.Abs(filepath.Join(basePath, filepath.FromSlash(source.Path)))
				if err != nil {
					return "", errors.Wrap(err, "filepath absolute")
				}
			}
		}
	}

	if source.SubPath != "" {
		localPath = filepath.Join(localPath, filepath.FromSlash(source.SubPath))
	}

	return getDependencyConfigPath(localPath, source)
}

func getDependencyConfigPath(dependencyPath string, source *latest.SourceConfig) (string, error) {
	var configPath string
	if source.ConfigName != "" {
		configPath = filepath.Join(dependencyPath, source.ConfigName)
	} else if strings.HasSuffix(dependencyPath, ".yaml") || strings.HasSuffix(dependencyPath, ".yml") {
		configPath = dependencyPath
		dependencyPath = filepath.Dir(dependencyPath)
	} else {
		configPath = filepath.Join(dependencyPath, constants.DefaultConfigPath)
	}

	return configPath, nil
}

func getDependencyID(source *latest.SourceConfig) (string, error) {
	// check if source is there
	if source == nil {
		return "", fmt.Errorf("source is missing")
	}

	// get id for git
	if source.Git != "" {
		id := source.Git
		if source.Branch != "" {
			id += "@" + source.Branch
		} else if source.Tag != "" {
			id += "@tag:" + source.Tag
		} else if source.Revision != "" {
			id += "@revision:" + source.Revision
		}

		return id, nil
	} else if source.Path != "" {
		return source.Path, nil
	}

	return "", fmt.Errorf("unexpected dependency config, both source.git and source.path are missing")
}

func GetParentProfileID(basePath string, source *latest.SourceConfig, profile string, vars []latest.DependencyVar) string {
	if source.Git != "" {
		// Erase authentication credentials
		id := strings.TrimSpace(source.Git)
		id = authRegEx.ReplaceAllString(id, "$1$2")

		if source.Tag != "" {
			id += "@" + source.Tag
		} else if source.Branch != "" {
			id += "@" + source.Branch
		} else if source.Revision != "" {
			id += "@" + source.Revision
		}
		if source.SubPath != "" {
			id += ":" + source.SubPath
		}
		if profile != "" {
			id += " - profile " + profile
		}
		if len(source.CloneArgs) > 0 {
			id += " - with clone args " + strings.Join(source.CloneArgs, " ")
		}
		for _, v := range vars {
			id += ";" + v.Name + "=" + v.Value
		}

		return id
	} else if source.Path != "" {
		if isURL(source.Path) {
			id := strings.TrimSpace(source.Path)

			if profile != "" {
				id += " - profile " + profile
			}
			for _, v := range vars {
				id += ";" + v.Name + "=" + v.Value
			}

			return id
		}

		// Check if it's an git repo
		filePath := source.Path
		if !filepath.IsAbs(source.Path) {
			filePath = filepath.Join(basePath, source.Path)
		}

		remote, err := git.GetRemote(filePath)
		if err == nil {
			return remote
		}

		if source.ConfigName != "" {
			filePath += filepath.Join(filePath, source.ConfigName)
		}

		if profile != "" {
			filePath += " - profile " + profile
		}

		for _, v := range vars {
			filePath += ";" + v.Name + "=" + v.Value
		}

		return filePath
	}

	return ""
}

func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}
