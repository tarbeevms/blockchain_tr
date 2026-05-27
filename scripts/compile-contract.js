const { execFileSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");
const sourcePath = path.join(root, "contracts", "Voting.sol");
const buildDir = path.join(root, "contracts", "build");
const backendBinPath = path.join(root, "backend", "contract", "Voting.bin");

const input = {
  language: "Solidity",
  sources: {
    "contracts/Voting.sol": {
      content: fs.readFileSync(sourcePath, "utf8")
    }
  },
  settings: {
    evmVersion: "london",
    outputSelection: {
      "*": {
        "*": ["abi", "evm.bytecode.object"]
      }
    }
  }
};

const npx = process.platform === "win32" ? "npx.cmd" : "npx";
const rawOutput = execFileSync(npx, ["solc@0.8.26", "--standard-json"], {
  input: JSON.stringify(input),
  encoding: "utf8",
  maxBuffer: 20 * 1024 * 1024
});

const jsonStart = rawOutput.indexOf("{");
if (jsonStart < 0) {
  throw new Error(`Unexpected solc output: ${rawOutput}`);
}

const output = JSON.parse(rawOutput.slice(jsonStart));
const errors = output.errors || [];
const fatal = errors.filter((item) => item.severity === "error");
if (fatal.length > 0) {
  for (const error of fatal) {
    console.error(error.formattedMessage || error.message);
  }
  process.exit(1);
}

const contract = output.contracts?.["contracts/Voting.sol"]?.Voting;
if (!contract) {
  throw new Error("Voting contract output was not produced");
}

fs.mkdirSync(buildDir, { recursive: true });
fs.writeFileSync(path.join(buildDir, "Voting.abi"), JSON.stringify(contract.abi, null, 2));
fs.writeFileSync(path.join(buildDir, "Voting.bin"), contract.evm.bytecode.object);
fs.writeFileSync(path.join(buildDir, "contracts_Voting_sol_Voting.abi"), JSON.stringify(contract.abi, null, 2));
fs.writeFileSync(path.join(buildDir, "contracts_Voting_sol_Voting.bin"), contract.evm.bytecode.object);
fs.writeFileSync(backendBinPath, contract.evm.bytecode.object);

console.log("Compiled Voting.sol for EVM London");
