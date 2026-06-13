package render

import "strings"

// CDN is a download provider. Template uses {owner} {repo} {branch} {path}.
type CDN struct {
	Name     string
	Template string
}

// URL fills the template placeholders. path is repo-relative, e.g.
// "mihomo/Google/Google_site.mrs".
func (c CDN) URL(owner, repo, branch, path string) string {
	return strings.NewReplacer(
		"{owner}", owner,
		"{repo}", repo,
		"{branch}", branch,
		"{path}", strings.TrimPrefix(path, "/"),
	).Replace(c.Template)
}
