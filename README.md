# ETH Tokyo Aug 2024

## Summary
We're building a silent payment solution that allows users on EVM chains to send payments to another person in completely private and anonymous way, meaning only the sender and recipient would know the details of the actual transaction.

Silent payment is something that is already executable on Bitcoin. For this to work on Ethereum, we are basically borrowing an open-source library that allows us to create a UTXO based privacy-preserving token using ZKP.

## Problem & opportunity
Transactions on EVM chains are completely open for everyone to see, so users can easily see what everyone else are doing/sending. However, there are certain situations where one may wanna send a payment to someone without letting the whole world know about it. For example, sending money to support a political dissident, or sending crypto to support an independent journalist.

## Solution
We have created an easy-to-use app that allows users to send payment to a recipient anonymously after logging in.

### Technology architecture and customer flow
![Blind Payments Architecture](Architecture.png)

## User Flow
User/sender will sign into the app with their Metamask wallet. From there:
1. User will make a deposit into the app
2. User will create a transfer so that the token will be allocated to a UTXO for the recipient
3. Only the intended recipient can withdraw and receive the donated token

## Future Work
Some thoughts for how the whole project can be built out further:
- Build out a more fully-fledged frontend
- Improve the whole process so that things can execute smoothly even in the browser or even on mobile
- Built things in a dynamic way so that it can take in any tokens from any users

## Deployed contract (all verified)
### Scroll Sepolia Testnet  (chainID: 534351)
| Contract      |                           Contract address | <br>
| ZeroToken     | 0x9f935aE3d93187755F0eBDD0D4db5F2F4b31b4Ee | <br>
| SampleERC20   | 0x133555dB8bD881890e3f86e6a39Ad622620BaB13 | <br>
The verified smart contract on Scroll Sepolia testnet is:
https://sepolia.scrollscan.com/address/0x9f935ae3d93187755f0ebdd0d4db5f2f4b31b4ee#code

### Linea Sepolia testnet (chainID: 59141)
| Contract      |                           Contract address | <br>
| ZeroToken     | 0x486098dE87321a5DBe79ACeAA10f5DC11E0fD2b7 | <br>
| SampleERC20   | 0xc82f8F12409Ed51f3d85b6350E4b96a97738D2C4 | <br>
The verified smart contract for Linea Sepolia testnet is:
https://sepolia.lineascan.build/address/0x486098dE87321a5DBe79ACeAA10f5DC11E0fD2b7#code

### Nero Chain testnet (chainID: 6660001)
| Contract      |                           Contract address |
| ZeroToken     |  |
| SampleERC20   |  |
The verified smart contract on Nero Chain testnet is:


### Arbitrum Sepolia testnet (chainID: 421614)
| Contract      |                           Contract address | <br>
| ZeroToken     | 0x46534b90d2aC25c70afFdfb37988592F5447b259 | <br>
| SampleERC20   | 0x133555dB8bD881890e3f86e6a39Ad622620BaB13 | <br>
The verified smart contract on Arbitrum Sepolia testnet is:
https://sepolia.arbiscan.io/address/0x46534b90d2ac25c70affdfb37988592f5447b259#code

## Others
### Demo movie


### Demo site


### How to run locally
Go to the backend and frontend folder, and then use this command:
1. npm install
2.
3.
