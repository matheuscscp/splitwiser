# splitwiser

A Telegram bot to help me parse my shared receipts and put the totals on https://www.splitwise.com/.

# Deployment steps

## Production

1. Install the the `jq` CLI for JSON manipulation (`sudo apt-get install jq`).
2. Install the Google Cloud SDK (the gcloud [CLI](https://cloud.google.com/sdk/docs/install)).
3. Run `gcloud auth login` to authenticate on a Google Cloud account.
4. Create a Google Cloud project (see `gcloud projects create --help`).
5. Set the new project as default: `gcloud config set project PROJECT_ID`.
6. Run `scripts/create-terraform-service-account.sh` to create a service account and a key JSON file for Terraform Cloud at `terraform-service-account-key.json`.
7. Create an organization and workspace in Terraform Cloud.
8. Configure the Terraform Cloud workspace with a VCS workflow connecting to your (fork) git repository.
9. Add a secret environment variable `GOOGLE_CREDENTIALS` with the minified JSON of the generated key file to the Terraform Cloud workspace.
10. Create all the manually-managed secrets in Google Cloud (the ones accessed in Terraform via the `google_secret_manager_secret_version` data source).
11. Run `scripts/enable-googleapis.sh` to enable the necessary Google Cloud APIs.
12. Open a pull request setting the new project ID, region and other options in `main.tf`.
13. Check out the Speculative Plan triggered by Terraform Cloud, it should be sent to you via email.
14. Merge and check out the Terraform Plan and Apply triggered by Terraform Cloud.
15. After the Terraform Apply finishes, go check the full IDs of all the secrets with a rotation policy in the Google Cloud Console (`projects/*/secrets/*`) and trigger the RotateSecret function with the JSON `{"attributes":{"secretId":"<secret-id>","eventType":"SECRET_ROTATE"}}` in the "Testing" tab of the Google Cloud Functions Console.
16. Verify that all the functions are working by checking out Google Cloud monitoring and testing tools in the Console.
17. Test the bot interaction: trigger StartBot via HTTP, type in the password, check hello Telegram message from bot.

## Development

The production deployment also creates development service accounts for each function so they can be tested locally under `cmd/<function>/` by running `go run .`.

1. Run `scripts/create-development-service-account-keys.sh` to create key JSON files for each function.
2. Craft the configuration file at `cmd/<function>/config.yml` for each function that needs one.
3. Run `cd cmd/<function>/` and `go run .` to test a function.

## Rotate Terraform-baked configuration secrets

If you need to rotate one of the secrets that is baked into a function configuration secret during Terraform Apply, trigger a Terraform Plan and Apply after making all the necessary rotations to update the configuration of the functions.
