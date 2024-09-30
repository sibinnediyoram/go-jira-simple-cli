package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	jira "github.com/andygrunwald/go-jira"
	"github.com/joho/godotenv"
)

type JiraService struct {
    Client *jira.Client
}

type Issue struct {
    Description string
    Type        string
    ProjectKey  string
    Summary     string
    Priority    string
    AssigneeID  string
}

func init() {
    if envErr := godotenv.Load(".env"); envErr != nil {
        fmt.Println(".env file missing")
    }
}
/*JIRA_TOKEN=<jira token can be generated here: https://id.atlassian.com/manage-profile/security/api-tokens>
JIRA_USER=<your email address>
JIRA_URL=<jira url can be gotten from the url (https://username.atlassian.net/jira/software/projects/<projectname>/boards) by extracting this (https://name.atlassian.net) only >
and feed these three in env variables
*/

func NewJiraService() *JiraService {
    jt := jira.BasicAuthTransport{
        Username: os.Getenv("JIRA_USER"),
        Password: os.Getenv("JIRA_TOKEN"),
    }

    client, err := jira.NewClient(jt.Client(), os.Getenv("JIRA_URL"))
    if err != nil {
        fmt.Println(err)
		os.Exit(1)
    }

    me, _, err := client.User.GetSelf()
    if err != nil {
        fmt.Println(err)
		os.Exit(1)
    }

    fmt.Println("Authenticated as:", me.DisplayName)
    return &JiraService{Client: client}
}

func (js *JiraService) CreateNewIssue(issue Issue, parentID string) string {
    i := jira.Issue{
        Fields: &jira.IssueFields{
            Description: issue.Description,
            Type: jira.IssueType{
                Name: issue.Type,
            },
            Project: jira.Project{
                Key: issue.ProjectKey,
            },
            Summary: issue.Summary,
            Priority: &jira.Priority{Name: issue.Priority},
    },
    }
    // Add Parent ID if it's a sub-task
    if parentID != "" {
        i.Fields.Parent = &jira.Parent{ID: parentID}
    }

     
    // Print the issue payload for debugging
    issuePayload, _ := json.MarshalIndent(i, "", "  ")
    fmt.Printf("Creating issue with payload: %s\n", issuePayload)

    createdIssue, res, err := js.Client.Issue.Create(&i)
    if err != nil {
        fmt.Println("Error creating issue:", err)
        body, _ := ioutil.ReadAll(res.Body)
        fmt.Println("Response body:", string(body))
        os.Exit(1)
    }

    // Print the created issue details
    fmt.Printf("Created issue: %+v\n", createdIssue)

    //update the assignee on the issue just created
    _, assignErr := js.Client.Issue.UpdateAssignee(createdIssue.ID, &jira.User{
        AccountID: issue.AssigneeID,
    })
//to get the assignee id, on your dashboard click on people.
//From the dropdown menu click on search people and teams and then click on the user you wish to assign a task to
//and then you should see this in ur url https://name.atlassian.net/jira/people/62exxxxxxxxxxxxxxxxxx7.
//The id <62exxxxxxxxxxxxxxxxxx7> after people endpoint is the assignee id

    if assignErr != nil {
        fmt.Println("Error assigning issue:", assignErr)
        os.Exit(1)
    }

    fmt.Println("Issue created:", createdIssue.Key)
    return createdIssue.ID
}

func promptForInput(promptText string) string {
    reader := bufio.NewReader(os.Stdin)
    fmt.Print(promptText)
    input, _ := reader.ReadString('\n')
    return strings.TrimSpace(input)
}

func main() {

    description := flag.String("description", "", "Description of the issue")
    issueType := flag.String("type", "", "Type of the issue (e.g., Bug, Task)")
    projectKey := flag.String("project", "", "Project key (e.g., MYPROJECT)")
    summary := flag.String("summary", "", "Summary of the issue")
    priority := flag.String("priority", "", "Priority of the issue (e.g., Low, Medium, High)")
    assigneeID := flag.String("assignee", "", "Assignee ID (e.g., 62exxxxxxxxxxxxxxxxxx7 for user sibin)") /* 62exxxxxxxxxxxxxxxxxx7 for user sibin */
    createSubtasks := flag.Bool("subtasks", false, "Create subtasks (true/false)")

    flag.Usage = func() {
        fmt.Printf("Usage of %s:\n", os.Args[0])
        flag.PrintDefaults()
        fmt.Println("\nEnvironment Variables:")
        fmt.Println("  JIRA_USER: Jira username (usually your email address)")
        fmt.Println("  JIRA_TOKEN: Jira API token (can be generated here: https://id.atlassian.com/manage-profile/security/api-tokens)")
        fmt.Println("  JIRA_URL: Jira URL (e.g., https://yourcompany.atlassian.net)")
    }

    flag.Parse()

    if *description == "" || *issueType == "" || *projectKey == "" || *summary == "" || *priority == "" || *assigneeID == "" {
        fmt.Println("All fields are required, if not sure check with --help")
        os.Exit(1)
    }

    issue := Issue{
        Description: *description,
        Type:        *issueType,
        ProjectKey:  *projectKey,
        Summary:     *summary,
        Priority:    *priority,
        AssigneeID:  *assigneeID,
    }

    jiraService := NewJiraService()
    issueID := jiraService.CreateNewIssue(issue, "")

    // Ask if user wants to create subtasks
    if *createSubtasks {
        for {
            subtaskDescription := promptForInput("Enter the description of the subtask: ")
            subtaskSummary := promptForInput("Enter the summary of the subtask: ")

            if subtaskDescription == "" || subtaskSummary == "" {
                fmt.Println("Description and Summary are required for subtasks")
                continue
            }

            subtask := Issue{
                Description: subtaskDescription,
                Type:        "Sub-task", // Sub-task type
                ProjectKey:  *projectKey,
                Summary:     subtaskSummary,
                Priority:    *priority, // Using the same priority as parent
                AssigneeID:  *assigneeID, // Using the same assignee as parent
            }

            jiraService.CreateNewIssue(subtask, issueID)

            moreSubtasks := promptForInput("Do you want to add another subtask? (yes/no): ")
            if strings.ToLower(moreSubtasks) != "yes" {
                break
            }
        }
    }
}
