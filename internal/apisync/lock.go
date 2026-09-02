// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
package apisync

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	exactVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	digestPattern       = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type ContractLock struct {
	SchemaVersion   int    `json:"schema_version"`
	ContractVersion string `json:"contract_version"`
	RegistryRef     string `json:"registry_ref"`
	PublishedSHA256 string `json:"published_sha256"`
}

type ContractCandidateExpectation struct {
	Version         string
	PublishedSHA256 string
}

type ContractCandidateStatus string

const (
	ContractCandidateCurrent ContractCandidateStatus = "current"
	ContractCandidateNew     ContractCandidateStatus = "new"
)

func InspectContractCandidate(lockPath, specPath string, expectation ContractCandidateExpectation) (ContractCandidateStatus, error) {
	lock, err := readContractLock(lockPath)
	if err != nil {
		return "", err
	}
	if err := validateContractLock(lock); err != nil {
		return "", err
	}
	spec, err := os.ReadFile(specPath) // #nosec G304: paths are explicit local workflow inputs.
	if err != nil {
		return "", fmt.Errorf("reading candidate contract: %w", err)
	}
	version, err := LoadSpecVersion(specPath)
	if err != nil {
		return "", err
	}
	if expectation.Version == "" || !exactVersionPattern.MatchString(expectation.Version) {
		return "", fmt.Errorf("expected contract version must be exact semver")
	}
	if version != expectation.Version {
		return "", fmt.Errorf("Scalar contract version %s does not match expected version %s", version, expectation.Version)
	}
	actualDigest := digest(spec)
	if expectation.PublishedSHA256 != "" {
		if !digestPattern.MatchString(expectation.PublishedSHA256) {
			return "", fmt.Errorf("expected published digest must be lowercase SHA-256")
		}
		if actualDigest != expectation.PublishedSHA256 {
			return "", fmt.Errorf("Scalar contract digest does not match publisher digest")
		}
	}

	switch compareExactVersions(version, lock.ContractVersion) {
	case -1:
		return "", fmt.Errorf("Scalar contract %s is older than pinned version %s", version, lock.ContractVersion)
	case 1:
		return ContractCandidateNew, nil
	default:
		if actualDigest != lock.PublishedSHA256 {
			return "", fmt.Errorf("Scalar contract %s changed bytes for an immutable version", version)
		}
		return ContractCandidateCurrent, nil
	}
}

func VerifyContractLock(lockPath, specPath string) (ContractLock, error) {
	lock, err := readContractLock(lockPath)
	if err != nil {
		return ContractLock{}, err
	}
	if err := validateContractLock(lock); err != nil {
		return ContractLock{}, err
	}
	spec, err := os.ReadFile(specPath) // #nosec G304: paths are explicit local workflow inputs.
	if err != nil {
		return ContractLock{}, fmt.Errorf("reading pinned contract: %w", err)
	}
	if digest(spec) != lock.PublishedSHA256 {
		return ContractLock{}, fmt.Errorf("pinned contract digest does not match contract lock")
	}
	version, err := LoadSpecVersion(specPath)
	if err != nil {
		return ContractLock{}, err
	}
	if version != lock.ContractVersion {
		return ContractLock{}, fmt.Errorf("pinned contract version does not match contract lock")
	}
	return lock, nil
}

func readContractLock(path string) (ContractLock, error) {
	data, err := os.ReadFile(path) // #nosec G304: paths are explicit local workflow inputs.
	if err != nil {
		return ContractLock{}, fmt.Errorf("reading contract lock %s: %w", path, err)
	}
	var lock ContractLock
	if err := decodeExactJSON(data, &lock); err != nil {
		return ContractLock{}, fmt.Errorf("parsing contract lock %s: %w", path, err)
	}
	return lock, nil
}

func decodeExactJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func validateContractLock(lock ContractLock) error {
	if lock.SchemaVersion != 1 {
		return fmt.Errorf("contract lock schema_version must be 1")
	}
	if !exactVersionPattern.MatchString(lock.ContractVersion) {
		return fmt.Errorf("contract version must be exact semver")
	}
	if lock.RegistryRef != "@straddle/straddle-api@"+lock.ContractVersion {
		return fmt.Errorf("registry reference does not match contract version")
	}
	if !digestPattern.MatchString(lock.PublishedSHA256) {
		return fmt.Errorf("contract lock digest must be lowercase SHA-256")
	}
	return nil
}

func compareExactVersions(left, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	for index := range leftParts {
		if len(leftParts[index]) < len(rightParts[index]) {
			return -1
		}
		if len(leftParts[index]) > len(rightParts[index]) {
			return 1
		}
		if leftParts[index] < rightParts[index] {
			return -1
		}
		if leftParts[index] > rightParts[index] {
			return 1
		}
	}
	return 0
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
