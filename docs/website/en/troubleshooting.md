# Troubleshooting: simple checks first

When something does not work, do these checks in order. Most issues are an incorrect URL, a stopped server, or a blocked port.

## 1. MCPBox does not start

| What you see | What to do |
| --- | --- |
| Port already in use | Choose another admin or MCP port, or stop the process using the port. |
| Cannot write database or files | Give the MCPBox service account read/write access to its application and knowledge-base folders. |

## 2. The admin page does not open

Confirm MCPBox is running. On the same server, open `http://127.0.0.1:38181/`. If that works but your own computer cannot open it, this is expected: the admin UI is private by default. Use a VPN, SSH tunnel, or approved reverse proxy instead of opening the port to the internet.

## 3. An MCP client cannot connect

Check the project MCP URL, including its project token. Then check whether the MCP server is enabled and healthy in the project. For an external client, confirm the HTTPS reverse proxy, firewall, and certificate. If bearer authentication or PolicyBox identity checking is enabled, make sure the client has the correct token.

For ChatGPT OAuth connections, inspect the `outcome` field in MCPBox **Logs**:

- `redirect_uri_mismatch` means the callback from the current ChatGPT setup does not match the stored value. Recreate the OAuth client with the callback from that same open ChatGPT window.
- `no such table: project_oauth_clients` means an older process is still running after the update. Restart MCPBox; startup will create the table or safely migrate it from the former name while preserving existing clients.

## 4. A Marketplace package fails

Read the server health error and Logs. Verify that the package’s runtime exists on the MCPBox server, its credentials are valid, and the service account can run its command. Remove a failed test package before retrying with different credentials.

## 5. Knowledge-base search is poor or empty

Check that the collection contains supported files and that text can actually be extracted. Scanned PDFs often need OCR before indexing. Reindex after changing source documents. Keep unrelated documents in separate collections and ask users specific questions that include useful names, dates, or terms.

## 6. RagBox or PolicyBox cannot be saved

Confirm the base URL is reachable from the MCPBox server, not only from your laptop. Recheck tenant ID and tokens, then use the integration status or runtime playground. Never paste a token into a public chat or a shared screenshot.

## What to send to support

Send the product version, operating system, time of the error, relevant log entry, and a description of the action that failed. Remove passwords, bearer tokens, API keys, and personal data first.
