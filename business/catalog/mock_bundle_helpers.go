package catalog

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"sort"
)

func buildMockTarball(artifact ReleaseArtifact) []byte {
	files := map[string]string{
		"package/SKILL.md":     buildMockSkillMD(artifact),
		"package/package.json": buildMockPackageJSON(artifact),
	}

	for _, component := range artifact.Version.Bundle.Components {
		switch component {
		case "mcp":
			files["package/.mcp.json"] = "{}"
		case "plugin":
			files["package/plugin.json"] = `{"name":"compat"}`
		default:
			files["package/"+component+"/placeholder.txt"] = "placeholder"
		}
	}

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		content := files[name]
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		_ = tarWriter.WriteHeader(header)
		_, _ = tarWriter.Write([]byte(content))
	}
	_ = tarWriter.Close()
	_ = gzipWriter.Close()
	return archive.Bytes()
}

func digestMockTarball(tarball []byte) (string, string) {
	sha512Sum := sha512.Sum512(tarball)
	sha1Sum := sha1.Sum(tarball)
	return "sha512-" + base64.StdEncoding.EncodeToString(sha512Sum[:]), hex.EncodeToString(sha1Sum[:])
}

func buildMockSkillMD(artifact ReleaseArtifact) string {
	return "---\nname: " + artifact.SkillDisplayName + "\ndescription: " + artifact.SkillDescription + "\n---\n"
}

func buildMockPackageJSON(artifact ReleaseArtifact) string {
	return `{"name":"` + artifact.Version.PackageName + `","version":"` + artifact.Version.Version + `"}`
}
