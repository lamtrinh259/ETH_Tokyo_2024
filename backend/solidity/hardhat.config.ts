// Copyright © 2024 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";
import "@openzeppelin/hardhat-upgrades";
import crypto from "crypto";
require("@nomicfoundation/hardhat-toolbox");
require("@nomicfoundation/hardhat-foundry");

const keys = [
  crypto.randomBytes(32).toString("hex"),
  crypto.randomBytes(32).toString("hex"),
  crypto.randomBytes(32).toString("hex"),
  crypto.randomBytes(32).toString("hex"),
  crypto.randomBytes(32).toString("hex"),
];

const config: HardhatUserConfig = {
  solidity: "0.8.20",
  paths: {
    sources: "contracts"
  },
  networks: {
    besu: {
      url: "http://localhost:8545",
      accounts: keys,
      gasPrice: 0,
    },
    scrollSepolia: {
      url:"https://scroll-sepolia.public.blastapi.io",
      accounts: [
        "f581e09146dc555821defda40505b319ace18e5262141258d1abb0ee1eb60dfe"
      ]
    }
  },
  etherscan: {
    apiKey: {
      scrollSepolia: "XN5VTWB95ZTFMNITS4KVWS29T1J62UFIJE",
    },
    customChains: [
      {
        network: 'scrollSepolia',
        chainId: 534351,
        urls: {
          apiURL: "https://api-sepolia.scrollscan.com/api",
          browserURL: 'https://sepolia.scrollscan.com/',
        }
      }
    ]
  }
};

export default config;
