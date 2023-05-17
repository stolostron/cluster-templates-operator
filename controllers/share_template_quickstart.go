package controllers

import (
	console "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ShareTemplateQuickStartName = "share-cluster-template"

func getNamespaceRoleBindingTask() console.ConsoleQuickStartTask {
	return console.ConsoleQuickStartTask{
		Title: "Configure the Namespace RoleBinding to [cluster-templates-user](k8s/cluster/clusterroles/cluster-templates-user) role",
		Description: `1. In the navigation menu, click [User Management]{{highlight qs-nav-usermanagement}}
2. Navigate to the **Roles** page.
3. Choose **cluster-templates-user** role from the list.
4. Go to the **RoleBindings** tab.
5. Click **Create RoleBinding**.
6. Enter a name, select a namespace, and enter a subject (user, group, or service account).
7. Click **Create**.
8. If needed, add more subjects using the YAML tab or the CLI.`,
		Review: &console.ConsoleQuickStartTaskReview{
			FailedTaskHelp: "This task isn’t verified yet. Try the task again.",
			Instructions: `Is the new RoleBinding configured?
1. In the navigation menu, click [User Management]{{highlight qs-nav-usermanagement}}.
2. Navigate to the **RoleBindings** page.
3. Check that the new RoleBinding is in the list.`,
		},
		Summary: &console.ConsoleQuickStartTaskSummary{
			Failed:  "Try the steps again.",
			Success: "You successfuly configured the namespace RoleBinding.",
		},
	}
}

func getClusterWideRoleBindingTask() console.ConsoleQuickStartTask {
	return console.ConsoleQuickStartTask{
		Title: "Configure the Cluster-wide RoleBinding to [cluster-templates-user-ct](k8s/cluster/clusterroles/cluster-templates-user-ct) role",
		Description: `1. In the navigation menu, click [User Management]{{highlight qs-nav-usermanagement}}.
2. Navigate to the **Roles** page.
3. Choose **cluster-templates-user-ct** role from the list.
4. Go to the RoleBindings tab.
5. Click **Create RoleBinding**.
6. Select the Binding type **Cluster-wide role binding**.
7. Enter a name and a subject (user, group, or service account).
8. Click **Create**.
9. If needed, add more subjects using the YAML tab or the CLI.`,
		Review: &console.ConsoleQuickStartTaskReview{
			FailedTaskHelp: "This task isn’t verified yet. Try the task again.",
			Instructions: `Is the new RoleBinding configured?
1. In the navigation menu, click [User Management]{{highlight qs-nav-usermanagement}}.
2. Navigate to the **RoleBindings** page.
3. Check that the new RoleBinding is in the list.`,
		},
		Summary: &console.ConsoleQuickStartTaskSummary{
			Failed:  "Try the steps again.",
			Success: "You successfully configured the Cluster-wide RoleBinding.",
		},
	}
}

func getShareTemplateQuickStartTasks() []console.ConsoleQuickStartTask {
	return []console.ConsoleQuickStartTask{getNamespaceRoleBindingTask(), getClusterWideRoleBindingTask()}
}

func GetShareTemplateQuickStart() *console.ConsoleQuickStart {
	return &console.ConsoleQuickStart{
		ObjectMeta: metav1.ObjectMeta{
			Name: ShareTemplateQuickStartName,
		},
		Spec: console.ConsoleQuickStartSpec{
			Conclusion:      "The ClusterTemplates can be used by the users with the configured permissions.",
			Description:     "Manage who can create a cluster from a template. ",
			DisplayName:     "Give access to a cluster template",
			Introduction:    `To enable unprivileged developers to create clusters, you’ll need to provide them with a namespace configured with the minimal required permissions for using templates. In addition, you’ll need to provide them the cluster-wide permissions to view ClusterTemplates.`,
			Icon:            `data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA3MjEuMTUgNzIxLjE1Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2RiMzkyNzt9LmNscy0ye2ZpbGw6I2NiMzYyODt9LmNscy0ze2ZpbGw6I2ZmZjt9LmNscy00e2ZpbGw6I2UzZTNlMjt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPlByb2R1Y3RfSWNvbi1SZWRfSGF0QWR2YW5jZWRfQ2x1c3Rlcl9NYW5hZ2VtZW50X2Zvcl9LdWJlcm5ldGVzLVJHQjwvdGl0bGU+PGcgaWQ9IkxheWVyXzEiIGRhdGEtbmFtZT0iTGF5ZXIgMSI+PGNpcmNsZSBjbGFzcz0iY2xzLTEiIGN4PSIzNjAuNTciIGN5PSIzNjAuNTciIHI9IjM1OC41OCIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTYxNC4xMywxMDcsMTA3LDYxNC4xM2MxNDAsMTQwLDM2Ny4wNywxNDAsNTA3LjExLDBTNzU0LjE2LDI0Ny4wNiw2MTQuMTMsMTA3WiIvPjxyZWN0IGNsYXNzPSJjbHMtMyIgeD0iMzMwLjg3IiB5PSIyODAuNiIgd2lkdGg9IjIwMy4xNyIgaGVpZ2h0PSIyMCIgdHJhbnNmb3JtPSJ0cmFuc2xhdGUoLTc4LjkgMzkwLjUyKSByb3RhdGUoLTQ0Ljk2KSIvPjxyZWN0IGNsYXNzPSJjbHMtMyIgeD0iMzA2LjYzIiB5PSIxNjcuODMiIHdpZHRoPSIyMCIgaGVpZ2h0PSIyMDQuNDciIHRyYW5zZm9ybT0idHJhbnNsYXRlKC04NS4zMyAxNjIuMjcpIHJvdGF0ZSgtMjUuNDUpIi8+PHJlY3QgY2xhc3M9ImNscy0zIiB4PSIxNjIuOTgiIHk9IjM2NC4xIiB3aWR0aD0iMTk4LjI4IiBoZWlnaHQ9IjIwIiB0cmFuc2Zvcm09InRyYW5zbGF0ZSgtNDIuMzkgMzMuNjEpIHJvdGF0ZSgtNi43OSkiLz48cmVjdCBjbGFzcz0iY2xzLTMiIHg9IjI0NS4xIiB5PSI0NTEuNTQiIHdpZHRoPSIyMDAuNjIiIGhlaWdodD0iMjAiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0xNjMuMDEgNzMzLjI2KSByb3RhdGUoLTgxLjMxKSIvPjxyZWN0IGNsYXNzPSJjbHMtMyIgeD0iNDQzLjg1IiB5PSIzMDMuNzYiIHdpZHRoPSIyMCIgaGVpZ2h0PSIyMDcuMDQiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0xMDkuOTcgNjM5LjU4KSByb3RhdGUoLTY0LjMpIi8+PGNpcmNsZSBjbGFzcz0iY2xzLTMiIGN4PSI1MDQuMzQiIGN5PSIyMTguODMiIHI9IjQ0LjA4Ii8+PGNpcmNsZSBjbGFzcz0iY2xzLTMiIGN4PSIyNzIuNyIgY3k9IjE3Ny43NSIgcj0iNDQuMDgiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMyIgY3g9IjU0Ny4xMiIgY3k9IjQ1Mi4xNyIgcj0iNDQuMDgiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMyIgY3g9IjE2My42OCIgY3k9IjM4NS44MiIgcj0iNDQuMDgiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMyIgY3g9IjMzMC4yNiIgY3k9IjU2MC43IiByPSI0NC4wOCIvPjxwYXRoIGNsYXNzPSJjbHMtNCIgZD0iTTQ0NC45NCwyNzkuOTIsMjc2LjE5LDQ0OC42N0ExMTkuMzIsMTE5LjMyLDAsMCwwLDQ0NC45NCwyNzkuOTJaIi8+PHBhdGggY2xhc3M9ImNscy0zIiBkPSJNMzc1LjY4LDI0NS43NmExMTkuMzMsMTE5LjMzLDAsMCwwLTk5LjQ5LDIwMi45MUw0NDQuOTQsMjc5LjkyQTExOC44OSwxMTguODksMCwwLDAsMzc1LjY4LDI0NS43NloiLz48L2c+PC9zdmc+`,
			DurationMinutes: 10,
			Tasks:           getShareTemplateQuickStartTasks(),
			NextQuickStart:  []string{QuotaQuickStartName},
		},
	}
}
