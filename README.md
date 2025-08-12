# 📒 cache-sync

[![License](https://img.shields.io/badge/license-AGPLv3-blue.svg)](LICENSE)
[![Build cache-sync (Manual Trigger)](https://github.com/haziqnorisham/cache-sync/actions/workflows/build-go.yml/badge.svg)](https://github.com/haziqnorisham/cache-sync/actions/workflows/build-go.yml)

Offline-First MQTT Data Caching for __*IAS*__ IIOT Reliability

---

## ✏️About

__*cache-sync*__ ensures zero data loss for your IIOT end-nodes by locally caching MQTT telemetry to an embedded SQLite database when internet connectivity is unavailable. Designed to guarentee __*IAS platform*__ data reliability, it asynchronously uploads cached data to __*IAS platform*__ server upon reconnection with unlimited retries, configurable queue sizes, and upload frequency controls.

Built for harsh environments: A lightweight, multithreaded Go service that guarantees data integrity where connectivity is unreliable.

## 🔑Key Features

- ✅ MQTT to SQLite Buffering – Listen to  end-node topics, cache data locally.
 
- ✅ Resilient Sync Engine – Automatic retries, configurable batch uploads.

- ✅ Thread-Safe & Low-Overhead – Multithreaded Go backend for high efficiency.

- ✅ Plug-and-Play for __*IAS*__ first-party hardware – Seamlessly extends your IIOT platform’s reliability

- ✅ Multi-Architecture Support – Runs on x86, x64, and ARM (Raspberry Pi, edge devices).

## 🖥️Screenshots

![alt text](cache-sync-console.png)

## 💽Binaries?

Head over to __*GitHub Actions*__ tab & open the last successfull workflow run. There should be where the pre-compiled binaries are located.

Binaries are only available for these platforms :
 - Linux
    - arm64/aarch64
    - amd64/x64

## 🏗️Deployment

### 🔧Prerequisite

__*cache-sync*__ will listen to an MQTT topic to cache & forward to central __*IAS*__ server endpoint for processing. With this in mind, we require an active MQTT server before we can start __*cache-sync*__. An existing external MQTT server can be used if already available.

### ⚠️Assumptions

It is assumed in this deplyment example, we are using a __*Ubuntu Server 24.04*__ installaion running on __*Intel amd64/x64*__ platform. The server specs used is as follows :

- 4 Core Intel Sandy Bridge Era Xeon
- 2GB DDR3 ECC RAM
- 25GB SATA SSD Storage

### 🔧Installing MQTT

Make sure everythin is up-to-date :
    
```bash
sudo apt update && sudo apt upgrade -y
```

Install __*mosquitto*__ MQTT broker/server :

```bash
sudo apt install mosquitto -y
```

Once installaiton is done, we can verify by checking __*mosquitto*__ version :

```bash
mosquitto -v
```

mousquitto should be running as a server & we can check that by running :

```bash
sudo systemctl status mosquitto
```

If everything goes well, we should see __*mosquitto*__ in ```active (running)``` status.

### ⚙️Configuring MQTT

By default, __*mosquitto*__ will load all files with ```.conf``` extension that is place in this directory: 

```bash
/etc/mosquitto/conf.d/
```

As a start we will create a new ```default.conf``` file to store our initial __*mosquitto*__ configuration :

```bash
sudo touch /etc/mosquitto/conf.d/default.conf
```

Now we will edit the ```default.conf``` file with our configurations :

```bash
sudo nano /etc/mosquitto/conf.d/default.conf
```

This is the contents of the ```default.conf``` config file : 

```bash
# Config file for mosquitto
# For lists of all supported configurations & examples, see :
#   -> /usr/share/doc/mosquitto/examples/mosquitto.conf
# -----------------------------------------------------------
# By default mosquitty only allows internal connections,
# 0.0.0.0 will allow connections from anywhere on port 1883
listener 1883 0.0.0.0

# Disable passwordless connection
allow_anonymous false

# Location of file that store user credentials
password_file /etc/mosquitto/passwd
```

Next, we need to configure the user credential file : 

```bash
sudo nano /etc/mosquitto/passwd
```

The format used is ```username:password```, here is an example of the contents of ```/etc/mosquitto/passwd``` file : 

```bash
cache-sync:changeme
client_02:passwd
```

Once username & password is configured in the file, we will encrypt the plain text password in that ```/etc/mosquitto/passwd``` file using:

```bash
sudo mosquitto_passwd -U /etc/mosquitto/passwd
```

Close & save the file, our configuration is now complete. All we need to do now is to restart the __*mosquitto*__ service :

```bash
sudo systemctl restart mosquitto
```

### ⚙️Configuring cache-sync

Download the application binary. Refer to ```💽Binaries?``` section for details. We can now create a new ```config.yaml``` in the same directory of the __*cache-sync*__ binary :

```bash
nano config.yaml
```

This is a sample content for ```config.yaml``` file :

```yaml
dev:
  database_url: postgres://postgres:@localhost:5432/database_dev
  mqtt_broker_address: 127.0.0.1
  mqtt_broker_port: 1883
  mqtt_broker_topic: application/
  mqtt_broker_user: cache-sync
  mqtt_broker_password: changeme
  uplink_endpoint: http://127.0.0.1:1880/cache-sync/uplink
prod:
  database_url: postgres://postgres:@localhost:5432/database_dev
  mqtt_broker_address: 127.0.0.1
  mqtt_broker_port: 1883
  mqtt_broker_topic: application/
  mqtt_broker_user: cache-sync
  mqtt_broker_password: changeme
  uplink_endpoint: http://127.0.0.1:1880/cache-sync/uplink
```

Here is an overview of the file structure for this program :

```bash
ubuntu@ubuntu:~/cache-sync$ ls -lrta
total 15452
-rwxrwxr-x 1 ubuntu ubuntu 15809833 Aug 12 01:20 cache-sync_amd64.bin
-rw-rw-r-- 1 ubuntu ubuntu      586 Aug 12 09:22 config.yaml
drwxr-x--- 7 ubuntu ubuntu     4096 Aug 12 09:24 ..
drwxrwxr-x 2 ubuntu ubuntu     4096 Aug 12 09:24 .
```

Our configuration is done. Finally, we give proper permission to the binary & run __*cache-sync*__ :

```bash
chmod +x ./cache-sync_amd64.bin
```
```bash
./cache-sync_amd64.bin
```

Application will start & we can see a simmilar output to the screenshot above.

## 📜License

__*cache-sync*__ is maintained by [haziqnorisham](https://github.com/haziqnorisham) for [Camart Sdn. Bhd.](https://camartcctv.com)

__*cache-sync*__ is licensed under the GNU Affero General Public License v3.0 (AGPLv3).
Key Points for Users:

- ✅ Freedom to Use, Modify, and Distribute – Provided you preserve license terms and disclose source code.

- 📜 Copyleft Requirement – Any derivative work or networked service using __*cache-sync*__ must also be open-sourced under AGPLv3.

- 🔗 Full License Text – See LICENSE file in this repository or read AGPLv3 on GNU.org.

---