# coldstorage

sia-coldstorage is a utility that generates a seed and a collection of addresses for a Sia wallet, without the blockchain. This is useful for creating 'cold wallets'; you can run this utility on an airgapped machine, write down the seed and the addresses, and send coins safely to any of the addresses without having to worry about malware stealing your coins. You can use Sia to restore the seed and spend the funds at any time.

## USAGE

The most basic usage generates a set of non-timelocked addresses requiring a single signature each. The addresses are all generated from one seed.

```
./coldstorage
```

Here is an example of generating a set of 3-of-5 multisig addresses timelocked to block 157680, with each address using pubkeys from 5 different seeds. The `-print` flag tells the command to print the pubkey info to the console, in addition to opening the web browser page with the address and seed info.

```
./coldstorage -timelock 157680 -n 3 -m 5 -print
```

If you would like to regenerate the addresses from a set of seeds, you can pass the seeds as follows:


```
./coldstorage -timelock 157680 -n 3 -m 5 -print -unique-seeds [seeds]
```

Please note that in the above the seeds, like the words in the seeds, should be just separated by spaces. The above command would need to be passed a list of 145 words (29 words each for 5 seeds), in the correct order, to regenerate the appropriate 5-pubkey addresses.


Ideally, you would run this on a system that was very secure, i.e. an airgapped
LiveCD. 




## LICENSE

The MIT License (MIT)
