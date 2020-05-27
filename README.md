## Open Source MintMe.com Coin Mining Pool

What follows next is the basic short guide. Main page of the project is located at https://mintme.com/coin. If you have a need to learn how to install in a more detailed way, we invite you to read our <b>[Guide on Wiki](https://github.com/webchain-network/webchain-pool/wiki/Detailed-guide-about-Webchain-pool-installation)</b>

### Features

* Support for Stratum mining
* Detailed block stats with luck percentage and full reward
* Failover webchaind instances: webchaind high availability built in
* Modern beautiful Ember.js frontend
* Separate stats for workers: can highlight timed-out workers so miners can perform maintenance of rigs
* JSON-API for stats
* Variable difficulty

### Building on Linux

Dependencies:

  * go >= 1.8
  * webchaind
  * redis-server >= 2.8.0
  * nodejs
  * nginx

**I highly recommend to use Ubuntu 16.04 LTS.**

First install  [webchaind](https://github.com/webchain-network/webchaind/releases).

Clone & compile:

    git clone https://github.com/webchain-network/webchain-pool.git
    cd webchain-pool
    make

Install redis-server.

### Running Pool

    ./build/bin/webchain-pool config.json

You can use Ubuntu upstart - check for sample config in <code>upstart.conf</code>.

### Building Frontend

Install nodejs. I suggest using LTS version >= 4.x from https://github.com/nodesource/distributions or from your Linux distribution.

The frontend is a single-page Ember.js application that polls the pool API to render miner stats.

    cd www

Change <code>ApiUrl: '//example.net/'</code> in <code>www/config/environment.js</code> to match your domain name. Also don't forget to adjust other options.

    npm install -g ember-cli@2.4.3
    npm install -g bower
    npm install
    bower install
    ./build.sh

Configure nginx to serve API on <code>/api</code> subdirectory.
Configure nginx to serve <code>www/dist</code> as static website.

#### Serving API using nginx

Add this setting after <code>location /</code>:

	location /api {
		if ($request_uri ~* "/api/(.*)") {
			proxy_pass  http://127.0.0.1:8080/apietc/$1;
		}
	}

#### Customization

You can customize the layout using built-in web server with live reload:

    ember server --port 8082 --environment development

**Don't use built-in web server in production**.

Check out <code>www/app/templates</code> directory and edit these templates
in order to customise the frontend.

### Configuration

Configuration is actually simple, just read it twice and think twice before changing defaults.

**Don't copy config directly from this manual. Use the example config from the package,
otherwise you will get errors on start because of JSON comments.**

```javascript
{
  // Set to the number of CPU cores of your server
  "threads": 2,
  // Prefix for keys in redis store
  "coin": "web",
  // Give unique name to each instance
  "name": "main",

  "proxy": {
    "enabled": true,

    // Bind HTTP mining endpoint to this IP:PORT
    "listen": "0.0.0.0:8888",

    // Allow only this header and body size of HTTP request from miners
    "limitHeadersSize": 1024,
    "limitBodySize": 256,

    /* Set to true if you are behind CloudFlare (not recommended) or behind http-reverse
      proxy to enable IP detection from X-Forwarded-For header.
      Advanced users only. It's tricky to make it right and secure.
    */
    "behindReverseProxy": false,

    // Stratum mining endpoint
    "stratum": {
      "enabled": true,
      // Bind stratum mining socket to this IP:PORT
      "listen": "0.0.0.0:8008",
      "timeout": "120s",
      "maxConn": 8192
    },

    // Variable share difficulty
    "varDiff": {
        // Minimum and maximum allowed difficulty. If you want to disable varDiff, set both keys to the same value
        "minDiff": 500,
        "maxDiff": 100000,
        // Adjust difficulty to get 1 share per N seconds
        "targetTime": 100,
        // Allow time to vary this percent from target without retargeting
        "variancePercent": 30,
        // Limit difficulty percent change in a single retargeting
        "maxJump": 50
    },

    // Try to get new job from webchaind in this interval
    "blockRefreshInterval": "120ms",
    "stateUpdateInterval": "3s",
    // Require this share difficulty from miners
    "difficulty": 1000,

    /* Reply error to miner instead of job if redis is unavailable.
      Should save electricity to miners if pool is sick and they didn't set up failovers.
    */
    "healthCheck": true,
    // Mark pool sick after this number of redis failures.
    "maxFails": 100,
    // TTL for workers stats, usually should be equal to large hashrate window from API section
    "hashrateExpiration": "3h",

    "policy": {
      "workers": 8,
      "resetInterval": "60m",
      "refreshInterval": "1m",

      "banning": {
        "enabled": false,
        /* Name of ipset for banning.
        Check http://ipset.netfilter.org/ documentation.
        */
        "ipset": "blacklist",
        // Remove ban after this amount of time
        "timeout": 1800,
        // Percent of invalid shares from all shares to ban miner
        "invalidPercent": 30,
        // Check after after miner submitted this number of shares
        "checkThreshold": 30,
        // Bad miner after this number of malformed requests
        "malformedLimit": 5
      },
      // Connection rate limit
      "limits": {
        "enabled": false,
        // Number of initial connections
        "limit": 30,
        "grace": "5m",
        // Increase allowed number of connections on each valid share
        "limitJump": 10
      }
    }
  },

  // Provides JSON data for frontend which is static website
  "api": {
    "enabled": true,
    "listen": "0.0.0.0:8080",
    // Collect miners stats (hashrate, ...) in this interval
    "statsCollectInterval": "5s",
    // Purge stale stats interval
    "purgeInterval": "10m",
    // Fast hashrate estimation window for each miner from it's shares
    "hashrateWindow": "30m",
    // Long and precise hashrate from shares, 3h is cool, keep it
    "hashrateLargeWindow": "3h",
    // Collect stats for shares/diff ratio for this number of blocks
    "luckWindow": [64, 128, 256],
    // Max number of payments to display in frontend
    "payments": 50,
    // Max numbers of blocks to display in frontend
    "blocks": 50,

    /* If you are running API node on a different server where this module
      is reading data from redis writeable slave, you must run an api instance with this option enabled in order to purge hashrate stats from main redis node.
      Only redis writeable slave will work properly if you are distributing using redis slaves.
      Very advanced. Usually all modules should share same redis instance.
    */
    "purgeOnly": false
  },

  // Check health of each webchaind node in this interval
  "upstreamCheckInterval": "5s",

  /* List of webchaind nodes to poll for new jobs. Pool will try to get work from
    first alive one and check in background for failed to back up.
    Current block template of the pool is always cached in RAM indeed.
  */
  "upstream": [
    {
      "name": "main",
      "url": "http://127.0.0.1:39573",
      "timeout": "10s"
    },
    {
      "name": "backup",
      "url": "http://127.0.0.2:39573",
      "timeout": "10s"
    }
  ],

  // This is standard redis connection options
  "redis": {
    // Where your redis instance is listening for commands
    "endpoint": "127.0.0.1:6379",
    "poolSize": 10,
    "database": 0,
    "password": ""
  },

  // This module periodically remits ether to miners
  "unlocker": {
    "enabled": false,
    // Pool fee percentage
    "poolFee": 1.0,
    // Pool fees beneficiary address (leave it blank to disable fee withdrawals)
    "poolFeeAddress": "",
    // Dev donation level (10.0 means 10% of poolFee, so if poolFee is set to 1.0 and devDonate to 10.0, dev donation level is 0.1%)
    "devDonate": 10.0,
    // Unlock only if this number of blocks mined back
    "depth": 120,
    // Simply don't touch this option
    "immatureDepth": 20,
    // Keep mined transaction fees as pool fees
    "keepTxFees": false,
    // Run unlocker in this interval
    "interval": "10m",
    // webchaind instance node rpc endpoint for unlocking blocks
    "daemon": "http://127.0.0.1:8545",
    // Rise error if can't reach webchaind in this amount of time
    "timeout": "10s"
  },

  // Pay out miners using this module
  "payouts": {
    "enabled": false,
    // Require minimum number of peers on node
    "requirePeers": 25,
    // Run payouts in this interval
    "interval": "12h",
    // webchaind instance node rpc endpoint for payouts processing
    "daemon": "http://127.0.0.1:39573",
    // Rise error if can't reach webchaind in this amount of time
    "timeout": "10s",
    // Address with pool balance
    "address": "0x0",
    // Let webchaind to determine gas and gasPrice
    "autoGas": true,
    // Gas amount and price for payout tx (advanced users only)
    "gas": "21000",
    "gasPrice": "50000000000",
    // Send payment only if miner's balance is >= 0.5 Ether
    "threshold": 500000000,
    // Perform BGSAVE on Redis after successful payouts session
    "bgsave": false
  }
}
```

If you are distributing your pool deployment to several servers or processes,
create several configs and disable unneeded modules on each server. (Advanced users)

I recommend this deployment strategy:

* Mining instance - 1x (it depends, you can run one node for EU, one for US, one for Asia)
* Unlocker and payouts instance - 1x each (strict!)
* API instance - 1x

### Notes

* Unlocking and payouts are sequential, 1st tx go, 2nd waiting for 1st to confirm and so on. You can disable that in code. Carefully read `docs/PAYOUTS.md`.
* Also, keep in mind that **unlocking and payouts will halt in case of backend or node RPC errors**. In that case check everything and restart.
* You must restart module if you see errors with the word *suspended*.
* Don't run payouts and unlocker modules as part of mining node. Create separate configs for both, launch independently and make sure you have a single instance of each module running.
* If `poolFeeAddress` is not specified all pool profit will remain on coinbase address. If it specified, make sure to periodically send some dust back required for payments.

### Credits
Ported to Webchain by Webchain project. Licensed under GPLv3.
Ported to Ethereum Classic by LeChuckDE, Licensed under GPLv3.
Made by sammy007. Licensed under GPLv3.

#### Contributors

[Alex Leverington](https://github.com/subtly)
