## xcp - put down the mouse!
You don't need to mouse to copy command result to clipboard. A very simple cli tool to setting of the X selection from stdin. 

<br />

### Description
It is very similar to [xclip](https://linux.die.net/man/1/xclip) without pasting/getting data from clipboard functionalities.

<br />

### Usage
Type your command and then use vertical bar character with __xcp__ (e.g: `ls -A1 | grep -E '\.sh|\.py' | xcp`) to copy previous command result to clipboard.
