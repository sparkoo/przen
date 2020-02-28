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

type conf struct {
    username string
    org      string
    repo     string
    prId     int
    spammer  string
    token    string
}

func main() {
    conf := parseArgs()
    client := ghClient(conf)
    defer printRateLimit(client)

    if pr, _, err := client.PullRequests.Get(context.Background(), conf.org, conf.repo, conf.prId); err != nil {
        log.Fatal(err)
    } else {
        //log.Printf("%+v", *pr.User)
        if *pr.User.Login != conf.username {
            log.Printf("username [%s] and author [%s] names dont match", *pr.User.Login, conf.username)
            log.Fatal("sorry, you can delete only comments on your own PRs")
        } else {
            log.Printf("usernames matches ok [%s]", conf.username)
        }

        //log.Printf("%+v", pr)
        fmt.Printf("[#%d - %s (%s)]\n", *pr.Number, *pr.Title, *pr.User.Login)
        fmt.Printf("[%s]\n", *pr.HTMLURL)

        confirm(client)
    }

    comments, _, err := client.Issues.ListComments(context.Background(), conf.org, conf.repo, conf.prId, &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: 100}})
    if err != nil {
        log.Fatal(err)
    }

    toDelete := make([]int64, 0)
    for _, comment := range comments {
        //log.Printf("%+v", comment)
        if *comment.User.Login == conf.spammer {
            log.Printf("comment [%d] by [%s] to delete", *comment.ID, *comment.User.Login)
            toDelete = append(toDelete, *comment.ID)
        }
    }

    log.Printf("[%d] comments to delete", len(toDelete))
    confirm(client)
    for _, commentId := range toDelete {
        log.Printf("about to delete comment [%d] ...", commentId)
        if _, deleteErr := client.Issues.DeleteComment(context.Background(), conf.org, conf.repo, commentId); deleteErr != nil {
            log.Fatal(deleteErr)
        } else {
            log.Print("ok")
        }
    }

}

func printRateLimit(client *github.Client) {
    if rates, _, err := client.RateLimits(context.Background()); err != nil {
        log.Fatal(err)
    } else {
        log.Printf("%+v", rates)
    }
}

func confirm(client *github.Client) {
    reader := bufio.NewReader(os.Stdin)
    fmt.Printf("ok? [y/n]: ")
    text, _ := reader.ReadString('\n')
    if text != "y\n" {
        printRateLimit(client)
        os.Exit(1)
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

    flag.StringVar(&conf.username, "username", "", "your github username, can be set with GITHUB_USERNAME env variable")
    flag.StringVar(&conf.org, "org", "", "name of the orgianization/user of the PR")
    flag.StringVar(&conf.repo, "repo", "", "name of the repo of the PR")
    flag.IntVar(&conf.prId, "prId", 0, "ID of the pull request")
    flag.StringVar(&conf.spammer, "spammer", "", "username of comments to delete")
    flag.StringVar(&conf.token, "token", "", "github token, can be set with GITHUB_TOKEN env var")

    flag.Parse()

    log.Printf("%+v", *conf)

    if conf.username == "" {
        if username, ok := os.LookupEnv("GITHUB_USERNAME"); ok {
            conf.username = username
        } else {
            log.Fatal("username must be set")
        }
    }

    if conf.token == "" {
        if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
            conf.token = token
        } else {
            log.Fatal("github token must be set")
        }
    }

    if conf.username == "" || conf.prId == 0 || conf.spammer == "" || conf.token == "" {
        log.Println("invalid args. usage:")
        flag.Usage()
        os.Exit(1)
    }

    return conf
}
