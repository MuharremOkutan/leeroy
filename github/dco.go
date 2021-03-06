package github

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/crosbymichael/octokat"
)

// DcoVerified checks if the pull request has been properly signed
func (g GitHub) DcoVerified(pr *PullRequest) (bool, error) {
	// we only want the prs that are opened/synchronized
	if !pr.Hook.IsOpened() && !pr.Hook.IsSynchronize() {
		return false, nil
	}

	// check if this is a bump branch, then we don't want to check sig
	if pr.ReleaseBase() {
		return true, nil
	}

	// we only want apply labels
	// to opened pull requests
	var labels []string

	//check if it's a proposal
	isProposal := strings.Contains(strings.ToLower(pr.Title), "proposal")
	switch {
	case isProposal:
		labels = []string{"status/1-design-review"}
	case pr.Content.IsNonCodeOnly():
		labels = []string{"status/3-docs-review"}
	default:
		labels = []string{"status/0-triage"}
	}

	// add labels if there are any
	// only add labels to new PRs not sync
	if len(labels) > 0 && pr.Hook.IsOpened() {
		logrus.Debugf("Adding labels %#v to pr %d", labels, pr.Hook.Number)

		if err := g.addLabel(pr.Repo, pr.Hook.Number, labels...); err != nil {
			return false, err
		}

		logrus.Infof("Added labels %#v to pr %d", labels, pr.Hook.Number)
	}

	var verified bool

	if pr.Content.CommitsSigned() {
		if err := g.removeLabel(pr.Repo, pr.Hook.Number, "dco/no"); err != nil {
			return false, err
		}

		if err := g.removeComment(pr.Repo, "sign your commits", pr.Content); err != nil {
			return false, err
		}

		if err := g.successStatus(pr.Repo, pr.Head.Sha, "docker/dco-signed", "All commits signed"); err != nil {
			return false, err
		}

		verified = true
	} else {
		if err := g.addLabel(pr.Repo, pr.Hook.Number, "dco/no"); err != nil {
			return false, err
		}

		if err := g.addDCOUnsignedComment(pr.Repo, pr, pr.Content); err != nil {
			return false, err
		}

		if err := g.failureStatus(pr.Repo, pr.Head.Sha, "docker/dco-signed", "Some commits without signature", "https://github.com/docker/docker/blob/master/CONTRIBUTING.md#sign-your-work"); err != nil {
			return false, err
		}
	}

	return verified, nil
}

func getRepo(repo *octokat.Repository) octokat.Repo {
	return octokat.Repo{
		Name:     repo.Name,
		UserName: repo.Owner.Login,
	}
}

func labelOs(pr *PullRequest, os string, fileChecker func() bool) bool {
	return strings.Contains(strings.ToLower(pr.Title), os) ||
		strings.Contains(strings.ToLower(pr.Body), os) ||
		fileChecker()
}
