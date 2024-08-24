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
import dotenv from 'dotenv';
dotenv.config();

const keys = [
  crypto.randomBytes(32).toString("hex"),
  crypto.randomBytes(32).toString("hex"),
  crypto.randomBytes(32).toString("hex"),
  crypto.randomBytes(32).toString("hex"),
  crypto.randomBytes(32).toString("hex"),
];
console.log("Private Key Loaded: ", process.env.PRIVATE_KEY ? "Yes" : "No");

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
    },
    nero: {
      url: "https://testnet.nerochain.io/",
      accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    },
    lineaSepolia: {
      url: "https://linea-sepolia.g.alchemy.com/v2/svFZH2X06Ju5hAy2Q_9aIAQ4QcJxd3ap",
      accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    },
    lineaGoerli: {
      url: "https://linea-goerli.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161",
      accounts: process.env.PRIVATE_KEY ? [process.env.PRIVATE_KEY] : [],
    }
  },
  etherscan: {
    apiKey: {
      scrollSepolia: process.env.SCROLL_SEPOLIASCAN_API_KEY || "",
      lineaSepolia: process.env.LINEA_SEPOLIA_API_KEY || "",
    },
    customChains: [
      {
        network: "scrollSepolia",
        chainId: 534351,
        urls: {
          apiURL: "https://api-sepolia.scrollscan.com/api",
          browserURL: "https://sepolia.scrollscan.com/",
        },
      },
      {
        network: "lineaSepolia",
        chainId: 59141,
        urls: {
          apiURL: "https://api-sepolia.lineascan.build/api",
          browserURL: "https://sepolia.lineascan.build/",
        },
      },
      {
        network: "lineaGoerli",
        chainId: 59140,
        urls: {
          apiURL: "https://explorer.goerli.linea.build/api",
          browserURL: "https://explorer.goerli.linea.build/",
        },
      }
    ]
  }
};

export default config;
