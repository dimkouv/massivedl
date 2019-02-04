# massivedl
Download a large list of files in parallel.

![](screenshots/output.gif)

## Install

```bash
# for linux 64bit
wget https://github.com/dimkouv/massivedl/releases/download/v1.0/massivedl_linux_x86_64
chmod +x massivedl_linux_x86_64
mv massivedl_linux_x86_64 /usr/local/bin/massivedl
```

## Usage

Create a .csv file with the downloads
```bash
filename,url
0.png,https://placehold.it/100x100
1.png,https://placehold.it/100x101
2.png,https://placehold.it/100x102
...
```

Assuming the file was named `data.csv` we can download the files using
```bash
massivedl -p 10 -i data.csv -s 1 -o downloads
```


### Command line parameters
```
-p <int> (default=10)          : Maximum number of parallel requests
-s <int> (default=0)           : Number of skipped lines from input csv
-i <str>                       : Input csv file with the list of urls
-o <str> (default='downloads') : Directory to place the downloads
```

### Stop and continue later
You can stop and continue downloading later.  
Press `Ctrl+C` then you will have the following dialog.

```bash
...
Do you want to save progress? [Y/n]: yes

Progress has been saved!
Use the following command to continue downloading

	massivedl --load /path/to/savedfile.save
```


## Use Cases
With this tool I was able to download about 1.5 million images (~60GB) for a machine learning project.
