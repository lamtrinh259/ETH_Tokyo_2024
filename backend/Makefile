verify_on_Scroll:
	forge verify-contract 0x133555dB8bD881890e3f86e6a39Ad622620BaB13 backend/solidity/contracts/erc20.sol:SampleERC20 --verifier-url https://api-sepolia.scrollscan.com/api --api-key XN5VTWB95ZTFMNITS4KVWS29T1J62UFIJE

verify_on_Linea_Sepolia:
	npx hardhat verify --network lineaSepolia 0xc82f8F12409Ed51f3d85b6350E4b96a97738D2C4 <initial_owner_param>

hardhat_verify:
	npx hardhat verify --network scrollSepolia 0x133555dB8bD881890e3f86e6a39Ad622620BaB13 <initial_owner_param>

deploy_on_Nero_testnet:
	ZETO_NAME=Zeto_Anon npx hardhat run scripts/deploy_upgradeable.ts --network nero

deploy_on_Linea_Sepolia:
	ZETO_NAME=Zeto_Anon npx hardhat run scripts/deploy_upgradeable.ts --network lineaSepolia
