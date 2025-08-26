# Remdit

## Demo Video

https://github.com/user-attachments/assets/6378574c-a315-4ac1-a144-43decc47212a

## Installation

```bash
curl -sSL https://raw.githubusercontent.com/krau/remdit/main/install.sh | sudo bash
```

## Usage

```bash
remdit yourfile.json
```

Then remdit will upload `yourfile.json` to a configured server (if none, upload to the default public server hosted by me), and return a edit link like `https://remdit.unv.app/uuid`. You can open the link in a browser to edit the file online. Changes will be synced back to your local file.

The edit link can be shared with others, so you can collaborate on the same file.

> Note:
> 
> I have no interest in your data. The default server is hosted on a free tier of a cloud provider, and the uploaded files will be deleted once the client disconnects.
> 
> If you want to be more secure, you can self-host your own server: https://github.com/krau/remdit-server

## Configuration

The configuration file is in TOML format. You can create a file named `config.toml` in one of the following locations:

- `/etc/remdit/config.toml`
- `$HOME/.remdit/config.toml`

Here is an example configuration file:

```toml
# config.toml
[[servers]]
addr = "https://remdit.unv.app"
key = ""

[[servers]]
addr = "https://your-own-server.com"
key = "your-api-key"
```

When multiple servers are configured, remdit will randomly choose one.

And also you can use environment variables to configure remdit, for example:

```bash
SERVERS_0_ADDR="https://your-own-server.com" \
SERVERS_0_KEY="your-api-key" \
remdit yourfile.json
```