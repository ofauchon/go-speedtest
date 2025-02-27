# go-speedtest

This is a simple tool to help you measure network speed. 

- You define a remote url for a file (--target http://www.somedomain.com/path/to/my/big/file)
- You define the concurrency (--concurrent 10 for 10 parallel downloads) 
- You can enable progress bars (--progress)

example: 

./go-speedtest --target http://somewhere.tld/my-big-file.data --concurrent 3 --progress


Warning: 

I wrote this tool to make tests and troubleshooting network at home.
The code was neither cleaned nor tested, don't trust it. 
