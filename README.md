# Personal GitHub Repo Manager CLI

This Go CLI application wraps several GitHub REST API endpoints to help manage your personal GitHub repositories. It provides functionalities to delete, open, archive, unarchive, make public, and make private your repositories. This tool is particularly useful for cleaning up personal GitHub accounts with numerous forks and repositories, as GitHub does not provide a way to manage repositories in batch.

## Features

- **Delete a repository**: Permanently delete a specified repository.
- **Open a repository**: Open the repository in the default web browser.
- **Archive a repository**: Archive a specified repository.
- **Unarchive a repository**: Unarchive a specified repository.
- **Make a repository public**: Change the visibility of a repository to public.
- **Make a repository private**: Change the visibility of a repository to private.
- **List repositories**: Fetch and list all repositories for the authenticated user.

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/ErvinRacz/personal-github-repo-manager.git
    cd personal-github-repo-manager
    ```
2. Obtain an API key from GitHub
3. Build the application:
    ```sh
    go run main.go --ghapikey=$(pass your_gh_api_key) --owner="<Your account name>"
    ```
