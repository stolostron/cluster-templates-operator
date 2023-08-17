package quickstarts

import (
	console "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getForkRepositoryTask() console.ConsoleQuickStartTask {
	return console.ConsoleQuickStartTask{
		Title: "Fork the community repository and create a GitHub page",
		Description: `1. Go to [github.com](http://www.github.com)
2. Log in to your account, or create a new one.
3. Go to the [community template repository](https://github.com/stolostron/cluster-templates-manifests) in GitHub.
4. Click the Fork button and then **Create a new fork**. 
5. Go to the git repository **Settings** tab and then click on the **Pages** on the side menu. Under **Branch** pick **main**, then click **Save**. Wait until the github pages are done generating (takes about a minute).`,
		Review: &console.ConsoleQuickStartTaskReview{
			FailedTaskHelp: "This task isn’t verified yet. Try the task again.",
			Instructions:   "Open `https://<github user name>.github.io/cluster-templates-manifests` in your browser. Do you see the repository README content?",
		},
		Summary: &console.ConsoleQuickStartTaskSummary{
			Failed:  "Try the steps again.",
			Success: "Your Helm Github page is available",
		},
	}
}

func getEditHelmChartTask() console.ConsoleQuickStartTask {
	//couldn't use multiline strings to create the description and instructions strings since golang doesn't support escaping backticks
	description := "1. Clone the repository from your terminal\n\n"
	description += "`git clone https://github.com/<github user name>/cluster-templates-manifests.git`\n\n"
	description += "2. Install the [Helm CLI](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.4/html/cli_tools/helm-cli) on your local machine.\n"
	description += "3. Edit one of the files in the hypershift-agent-template folder.\n"
	instructions := "1. Verify the chart passes lint.\n\n"
	instructions += "`helm lint hypershift-agent-template`\n\n"
	instructions += "2. Verify the chart contains the expected resources.\n\n"
	instructions += "`helm template hypershift-agent-template`\n\n"
	instructions += "Does the chart contain the expected Kubernetes resources?"
	return console.ConsoleQuickStartTask{
		Title:       "Edit the Helm chart",
		Description: description,
		Review: &console.ConsoleQuickStartTaskReview{
			FailedTaskHelp: "This task isn’t verified yet. Try the task again.",
			Instructions:   instructions,
		},
		Summary: &console.ConsoleQuickStartTaskSummary{
			Failed:  "Try the steps again.",
			Success: "Your Helm chart has been updated successfully.",
		},
	}
}

func getPackageAndPushTask() console.ConsoleQuickStartTask {
	description := "1. Package the chart into a versioned archived file.\n\n"
	description += "`helm package hypershift-agent-template --version \"0.0.2\"`\n\n"
	description += "2. Update the helm repository index file.\n\n"
	description += "`helm repo index . --url https://<github user name>.github.io/cluster-templates-manifests`\n\n"
	description += "3. Push the changes.\n\n"
	description += "`git add -A && git commit -m changes && git push`"
	instructions := "Verify that the URL provides the Helm index including the new chart version - It might take a few minutes until GitHub publishes the changes.\n\n"
	instructions += "`curl https://<github user name>.github.io/cluster-templates-manifests/index.yaml`"
	return console.ConsoleQuickStartTask{
		Title:       "Package and and push the changes",
		Description: description,
		Review: &console.ConsoleQuickStartTaskReview{
			FailedTaskHelp: "This task isn’t verified yet. Try the task again.",
			Instructions:   instructions,
		},
		Summary: &console.ConsoleQuickStartTaskSummary{
			Failed:  "Try the steps again.",
			Success: "Your helm repository is ready.",
		},
	}
}

func getCreateTemplateTask() console.ConsoleQuickStartTask {
	description := `1. From the **All Clusters** perspective navigate to the [Cluster Templates](k8s/cluster/clustertemplate.openshift.io~v1alpha1~ClusterTemplate) page by clicking **Infrastructure > Cluster templates**.
2. Click **Create a template**, fill in the details about the template in the first step of the wizard and click **Next**.
3. In the installation step, click on the **Helm chart repository** dropdown, and then click on **Add another repository URL**.`
	description += "\n4. Enter `https://<github user name>.github.io/cluster-templates-manifests` to the **Repository URL** field and click **Add**.\n"
	description += `5. Select the **hypershift-agent-template** chart and click **Next**.
6. Optionally define ArgoCD Applications to install on the clusters created from the template and click **Next**..
7. Click on **Create template**.`
	return console.ConsoleQuickStartTask{
		Title:       "Add a new cluster template",
		Description: description,
		Review: &console.ConsoleQuickStartTaskReview{
			FailedTaskHelp: "This task isn’t verified yet. Try the task again.",
			Instructions: `Verify that the template was created successfully.  
Is the status of the template **Ready**?`,
		},
		Summary: &console.ConsoleQuickStartTaskSummary{
			Failed:  "Try the steps again.",
			Success: "You have successfully created a cluster template.",
		},
	}
}

func getCreateClusterTask() console.ConsoleQuickStartTask {
	description := `1. Select **Create a cluster** from the **kebab menu** of the new template.
2. Enter a name.
3. Select the **clusters** namespace, which was created during the prerequisite setup.
4. Wait until the cluster status is **Ready**.`
	return console.ConsoleQuickStartTask{
		Title:       "Try the template",
		Description: description,
		Review: &console.ConsoleQuickStartTaskReview{
			FailedTaskHelp: "This task isn’t verified yet. Try the task again.",
			Instructions: `Verify that you can login to the cluster.
1. Select **Show credentials** from the **kebab menu** of the cluster.
2. Click **Copy login command**.
3. Run the command in a terminal.
4. Did you manage to login successfully?`,
		},
		Summary: &console.ConsoleQuickStartTaskSummary{
			Failed:  "Try the steps again.",
			Success: "You have successfully created a cluster from the template.",
		},
	}
}

func getTemplateQuickStartTasks() []console.ConsoleQuickStartTask {
	return []console.ConsoleQuickStartTask{getForkRepositoryTask(), getEditHelmChartTask(), getPackageAndPushTask(), getCreateTemplateTask(), getCreateClusterTask()}
}

func GetTemplateQuickStart() *console.ConsoleQuickStart {
	return &console.ConsoleQuickStart{
		ObjectMeta: metav1.ObjectMeta{
			Name: "create-cluster-template",
		},
		Spec: console.ConsoleQuickStartSpec{
			Conclusion:  "You successfully created your cluster template.",
			Description: "Use your repository to create your own cluster template that can be used to easily create clusters with the same configurations.",
			DisplayName: "Create a cluster template from scratch",
			Introduction: `By creating a cluster template, you’ll be able to quickly create clusters with the same configurations, such as clusters for test environments.  
In this quick start you will create a template based on the [hypershift-agent-cluster](k8s/cluster/clustertemplate.openshift.io~v1alpha1~ClusterTemplate/hypershift-agent-cluster) community one.  
Before you proceed, ensure that you set up the prerequisites, which can be found in the **Template description** section of the [hypershift-agent-cluster](k8s/cluster/clustertemplate.openshift.io~v1alpha1~ClusterTemplate/hypershift-agent-cluster) template page.`,
			Icon:            `data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA3MjEuMTUgNzIxLjE1Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2RiMzkyNzt9LmNscy0ye2ZpbGw6I2NiMzYyODt9LmNscy0ze2ZpbGw6I2ZmZjt9LmNscy00e2ZpbGw6I2UzZTNlMjt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPlByb2R1Y3RfSWNvbi1SZWRfSGF0QWR2YW5jZWRfQ2x1c3Rlcl9NYW5hZ2VtZW50X2Zvcl9LdWJlcm5ldGVzLVJHQjwvdGl0bGU+PGcgaWQ9IkxheWVyXzEiIGRhdGEtbmFtZT0iTGF5ZXIgMSI+PGNpcmNsZSBjbGFzcz0iY2xzLTEiIGN4PSIzNjAuNTciIGN5PSIzNjAuNTciIHI9IjM1OC41OCIvPjxwYXRoIGNsYXNzPSJjbHMtMiIgZD0iTTYxNC4xMywxMDcsMTA3LDYxNC4xM2MxNDAsMTQwLDM2Ny4wNywxNDAsNTA3LjExLDBTNzU0LjE2LDI0Ny4wNiw2MTQuMTMsMTA3WiIvPjxyZWN0IGNsYXNzPSJjbHMtMyIgeD0iMzMwLjg3IiB5PSIyODAuNiIgd2lkdGg9IjIwMy4xNyIgaGVpZ2h0PSIyMCIgdHJhbnNmb3JtPSJ0cmFuc2xhdGUoLTc4LjkgMzkwLjUyKSByb3RhdGUoLTQ0Ljk2KSIvPjxyZWN0IGNsYXNzPSJjbHMtMyIgeD0iMzA2LjYzIiB5PSIxNjcuODMiIHdpZHRoPSIyMCIgaGVpZ2h0PSIyMDQuNDciIHRyYW5zZm9ybT0idHJhbnNsYXRlKC04NS4zMyAxNjIuMjcpIHJvdGF0ZSgtMjUuNDUpIi8+PHJlY3QgY2xhc3M9ImNscy0zIiB4PSIxNjIuOTgiIHk9IjM2NC4xIiB3aWR0aD0iMTk4LjI4IiBoZWlnaHQ9IjIwIiB0cmFuc2Zvcm09InRyYW5zbGF0ZSgtNDIuMzkgMzMuNjEpIHJvdGF0ZSgtNi43OSkiLz48cmVjdCBjbGFzcz0iY2xzLTMiIHg9IjI0NS4xIiB5PSI0NTEuNTQiIHdpZHRoPSIyMDAuNjIiIGhlaWdodD0iMjAiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0xNjMuMDEgNzMzLjI2KSByb3RhdGUoLTgxLjMxKSIvPjxyZWN0IGNsYXNzPSJjbHMtMyIgeD0iNDQzLjg1IiB5PSIzMDMuNzYiIHdpZHRoPSIyMCIgaGVpZ2h0PSIyMDcuMDQiIHRyYW5zZm9ybT0idHJhbnNsYXRlKC0xMDkuOTcgNjM5LjU4KSByb3RhdGUoLTY0LjMpIi8+PGNpcmNsZSBjbGFzcz0iY2xzLTMiIGN4PSI1MDQuMzQiIGN5PSIyMTguODMiIHI9IjQ0LjA4Ii8+PGNpcmNsZSBjbGFzcz0iY2xzLTMiIGN4PSIyNzIuNyIgY3k9IjE3Ny43NSIgcj0iNDQuMDgiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMyIgY3g9IjU0Ny4xMiIgY3k9IjQ1Mi4xNyIgcj0iNDQuMDgiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMyIgY3g9IjE2My42OCIgY3k9IjM4NS44MiIgcj0iNDQuMDgiLz48Y2lyY2xlIGNsYXNzPSJjbHMtMyIgY3g9IjMzMC4yNiIgY3k9IjU2MC43IiByPSI0NC4wOCIvPjxwYXRoIGNsYXNzPSJjbHMtNCIgZD0iTTQ0NC45NCwyNzkuOTIsMjc2LjE5LDQ0OC42N0ExMTkuMzIsMTE5LjMyLDAsMCwwLDQ0NC45NCwyNzkuOTJaIi8+PHBhdGggY2xhc3M9ImNscy0zIiBkPSJNMzc1LjY4LDI0NS43NmExMTkuMzMsMTE5LjMzLDAsMCwwLTk5LjQ5LDIwMi45MUw0NDQuOTQsMjc5LjkyQTExOC44OSwxMTguODksMCwwLDAsMzc1LjY4LDI0NS43NloiLz48L2c+PC9zdmc+`,
			DurationMinutes: 10,
			Tasks:           getTemplateQuickStartTasks(),
			NextQuickStart:  []string{ShareTemplateQuickStartName, QuotaQuickStartName},
		},
	}
}
