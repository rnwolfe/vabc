# vabc demo — render with: vhs demo/vabc.tape
# Requires vabc on PATH (this tape prepends the repo dir).
Output demo/vabc.gif

Set FontSize 17
Set Width 1180
Set Height 700
Set Padding 22
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms

Hide
Type "export PATH=$PWD:$PATH; clear"
Enter
Show

Sleep 600ms
Type "# find a hard-to-get bottle (live web catalog)"
Enter
Type "vabc product search oftd --select productCode,name,size,retailPrice"
Enter
Sleep 3s

Type "# who has it in stock near me?"
Enter
Type "vabc inventory check 953714 --near 22182 --json | jq '[.nearbyStores[] | {store:.storeNumber, city, qty:.quantity, mi:.distance}]'"
Enter
Sleep 4s

Type "vabc store near 22182 --limit 4 --select storeNumber,city,distance"
Enter
Sleep 4s
