# gcp-update-firewall
Go tool to update GCP firewall rules which are open to the world (0.0.0.0/0) to your external public IP.

The -p STRING flag allows you to match firewall rule name that contains the STRING anywhere within the name.

The tool will exit if there are no rules to update, and will always as BEFORE updating any rules.

## Usage

```
Usage of ./gcp-update-firewall:
  -f    Force the updating of rules where the IP is set already to a single host
  -j string
        Path to Google auth json (If not specified default location will be assumed)
  -p string
        Pattern to match when scanning firewall rule names (* not needed matches on string found anywhere in name)
```

## Example

```
$ ./gcp-update-firewall -j ~/account.json -p jh                                                                           
                                                                                                                                                              
Firewall rules will be updated with the external IP: 1.1.1.1/32                                                                                               
                                                                                                                                                              
Not updating         Reason                                                                                                                        
------------         ------                                                                                                                        
gcp-test-node-e      Uses SourceTags [bastion]

Going to update the following rules:

gcp-test-node-a
gcp-test-node-b
gcp-test-node-c
gcp-test-node-d

Are you sure you want to update the above rules? (y/n): y

Updated              IP              OLD IP
-------              --              ------
gcp-test-node-a      1.1.1.1/32      0.0.0.0/0
gcp-test-node-b      1.1.1.1/32      0.0.0.0/0
gcp-test-node-c      1.1.1.1/32      0.0.0.0/0
gcp-test-node-d      1.1.1.1/32      0.0.0.0/0
```