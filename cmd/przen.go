package main

import (
    "bufio"
    "context"
    "flag"
    "fmt"
    "github.com/google/go-github/v29/github"
    "golang.org/x/oauth2"
    "log"
    "os"
)

func main() {
    conf := parseArgs()
    client := ghClient(conf)

    if pr, response, err := client.PullRequests.Get(context.Background(), conf.org, conf.repo, conf.prId); err != nil {
        log.Fatal(err)
    } else {
        if *pr.User.Login != conf.username {
            log.Printf("username [%s] and author [%s] names dont match", *pr.User.Login, conf.username)
            log.Fatal("sorry, you can delete only comments on your own PRs")
        } else {
            log.Printf("usernames matches ok [%s]", conf.username)
        }

        log.Printf("%+v", response)
        //log.Printf("%+v", pr)
        fmt.Printf("[#%d - %s (%s)]\n", *pr.Number, *pr.Title, *pr.User.Login)
        fmt.Printf("[%s]\n", *pr.HTMLURL)

        confirm()
    }

    comments, response, err := client.Issues.ListComments(context.Background(), conf.org, conf.repo, conf.prId, &github.IssueListCommentsOptions{})
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("%+v", response)

    toDelete := make([]int64, 0)
    for _, comment := range comments {
        //log.Printf("%+v", comment)
        if *comment.User.Login == conf.spammerUsername {
            log.Printf("comment [%d] to delete", *comment.ID)
            toDelete = append(toDelete, *comment.ID)
        }
    }

    log.Printf("[%d] comments to delete", len(toDelete))
    confirm()
    for _, commentId := range toDelete {
        log.Printf("deleting comment [%d]", commentId)
        if r, deleteErr := client.Issues.DeleteComment(context.Background(), conf.org, conf.repo, commentId); deleteErr != nil {
            log.Fatal(deleteErr)
        } else {
            log.Printf("%+v", r)
        }
    }

}

func confirm() {
    reader := bufio.NewReader(os.Stdin)
    fmt.Printf("ok? [y/n]: ")
    text, _ := reader.ReadString('\n')
    if text != "y\n" {
        log.Fatal("interrupted")
    }
}

func ghClient(conf *conf) *github.Client {

    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: conf.token},
    )
    tc := oauth2.NewClient(ctx, ts)

    return github.NewClient(tc)
}

func parseArgs() *conf {
    var conf = &conf{}

    flag.StringVar(&conf.username, "username", "", "your github username")
    flag.StringVar(&conf.org, "org", "", "name of the orgianization/user of the PR")
    flag.StringVar(&conf.repo, "repo", "", "name of the repo of the PR")
    flag.IntVar(&conf.prId, "prId", 0, "ID of the pull request")
    flag.StringVar(&conf.spammerUsername, "spammerUsername", "", "username of comments to delete")
    flag.StringVar(&conf.token, "token", "", "github token")

    flag.Parse()

    fmt.Printf("%+v\n", *conf)

    if conf.username == "" || conf.prId == 0 || conf.spammerUsername == "" || conf.token == "" {
        log.Fatal("You must define all params. Use `--help`.")
    }

    return conf
}

type conf struct {
    username        string
    org             string
    repo            string
    prId            int
    spammerUsername string
    token           string
}
