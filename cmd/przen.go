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
    "strconv"
    "strings"
)

type conf struct {
    username string
    owner    string
    repo     string
    prId     int
    spammer  string
    token    string
}

func main() {
    conf := parseArgs()
    client := ghClient(conf)
    defer printRateLimit(client)

    ensurePrId(client, conf)
    prConfirm(client, conf)
    toDelete := listComments(client, conf)
    deleteComments(client, conf, toDelete)
}

func deleteComments(client *github.Client, conf *conf, toDelete []int64) {
    if len(toDelete) <= 0 {
        fmt.Printf("nothing to delete here ...\n")
        printRateLimit(client)
        os.Exit(0)
    }

    fmt.Printf("%d comments to delete\n", len(toDelete))
    confirm(client)
    for _, commentId := range toDelete {
        fmt.Printf("about to delete comment [%d] ... ", commentId)
        if _, deleteErr := client.Issues.DeleteComment(context.Background(), conf.owner, conf.repo, commentId); deleteErr != nil {
            fmt.Print("fail\n")
            log.Fatal(deleteErr)
        } else {
            fmt.Print("ok\n")
        }
    }
    fmt.Println()
}

func listComments(client *github.Client, conf *conf) []int64{
    comments, _, err := client.Issues.ListComments(context.Background(), conf.owner, conf.repo, conf.prId, &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: 100}})
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println()
    toDelete := make([]int64, 0)
    for _, comment := range comments {
        //fmt.Printf("%+v", comment)
        if *comment.User.Login == conf.spammer {
            fmt.Printf("comment [%d] by [%s] to delete\n", *comment.ID, *comment.User.Login)
            toDelete = append(toDelete, *comment.ID)
        }
    }
    return toDelete
}

func prConfirm(client *github.Client, conf *conf) {
    fmt.Printf("checking if username [%s] matches ... ", conf.username)
    if pr, _, err := client.PullRequests.Get(context.Background(), conf.owner, conf.repo, conf.prId); err != nil {
        log.Fatal(err)
    } else {
        //fmt.Printf("%+v", *pr.User)
        if *pr.User.Login != conf.username {
            fmt.Printf("username [%s] and author [%s] names dont match", *pr.User.Login, conf.username)
            log.Fatal("sorry, you can delete only comments on your own PRs")
        } else {
            fmt.Printf("ok\n")
        }

        //fmt.Printf("%+v", pr)
        fmt.Printf("\n(#%d) %s\n", *pr.Number, *pr.Title)
        fmt.Printf("%s\n", *pr.HTMLURL)

        confirm(client)
    }
}

func ensurePrId(client *github.Client, conf *conf) {
    fmt.Println()
    if conf.prId == 0 {
        fmt.Printf("listing %s's PRs ... ", conf.username)
        usersPRs := listUsersPRs(client, conf)
        fmt.Println("ok")
        fmt.Println()
        for i, pr := range usersPRs {
            fmt.Printf("%d] (#%d) %s \n", i, *pr.Number, *pr.Title)
        }
        if prI, converr := strconv.Atoi(readInput("choose PR")); converr != nil {
            log.Fatal(converr)
        } else {
            conf.prId = *usersPRs[prI].Number
        }
        fmt.Println()
    }
}

func listUsersPRs(client *github.Client, c *conf) []github.PullRequest {
    prs, _, err := client.PullRequests.List(context.Background(), c.owner, c.repo, &github.PullRequestListOptions{
        ListOptions: github.ListOptions{
            PerPage: 1000,
        },
    })
    usersPRs := make([]github.PullRequest, 0)
    for _, pr := range prs {
        if *pr.User.Login == c.username {
            usersPRs = append(usersPRs, *pr)
        }
    }
    if err != nil {
        log.Fatal(err)
    }
    return usersPRs
}

func printRateLimit(client *github.Client) {
    fmt.Println("\nlisting github rates:")
    if rates, _, err := client.RateLimits(context.Background()); err != nil {
        log.Fatal(err)
    } else {
        fmt.Printf("%+v\n", rates)
    }
}

func confirm(client *github.Client) {
    if readInput("ok? [y/n]") != "y" {
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
    flag.StringVar(&conf.owner, "owner", "", "name of the owner of the repo of the PR")
    flag.StringVar(&conf.repo, "repo", "", "name of the repo of the PR")
    flag.IntVar(&conf.prId, "prId", 0, "ID of the pull request")
    flag.StringVar(&conf.spammer, "spammer", "", "username of comments to delete")
    flag.StringVar(&conf.token, "token", "", "github token, can be set with GITHUB_TOKEN env var")

    flag.Parse()

    fmt.Printf("%+v\n", *conf)

    if conf.username == "" {
        if username, ok := os.LookupEnv("GITHUB_USERNAME"); ok {
            fmt.Println("found github username from GITHUB_USERNAME env")
            conf.username = username
        } else {
            if conf.username = readInput("your username"); conf.username == "" {
                log.Fatal("username can't be empty")
            }
        }
    }

    if conf.token == "" {
        if token, ok := os.LookupEnv("GITHUB_TOKEN"); ok {
            fmt.Println("found github token from GITHUB_TOKEN env")
            conf.token = token
        } else {
            log.Fatal("github token must be set")
        }
    }

    if conf.owner == "" {
        if conf.owner = readInput("GH Repo owner"); conf.owner == "" {
            log.Fatal("owner can't be empty")
        }
    }

    if conf.repo == "" {
        if conf.repo = readInput("GH Repo"); conf.repo == "" {
            log.Fatal("repo can't be empty")
        }
    }

    if conf.spammer == "" {
        if conf.spammer = readInput("spammer username"); conf.spammer == "" {
            log.Fatal("spammer can't be empty")
        }
    }

    return conf
}

func readInput(prompt string) string {
    reader := bufio.NewReader(os.Stdin)
    fmt.Printf("%s: ", prompt)
    text, err := reader.ReadString('\n')
    if err != nil {
        log.Fatal(err)
    }
    return strings.Trim(text, "\n")
}
