#!/usr/bin/env bun
/**
 * Transparency Service CLI
 * Main entry point for all CLI commands
 */

import { error, usage } from "./utils/output.ts";

/**
 * Parse CLI arguments
 */
function parseArgs(args: string[]): { command: string; subcommand?: string; options: Record<string, any> } {
  const [command, subcommand, ...rest] = args;

  const options: Record<string, any> = {};

  for (let i = 0; i < rest.length; i++) {
    const arg = rest[i]!;

    if (arg.startsWith("--")) {
      const key = arg.slice(2);
      const nextArg = rest[i + 1];

      if (nextArg && !nextArg.startsWith("--")) {
        // Parse value
        options[key] = nextArg;
        i++;
      } else {
        // Boolean flag
        options[key] = true;
      }
    }
  }

  return { command, subcommand, options };
}

/**
 * Main CLI handler
 */
async function main() {
  const args = process.argv.slice(2);

  if (args.length === 0) {
    printHelp();
    process.exit(0);
  }

  const { command, subcommand, options } = parseArgs(args);

  try {
    // Handle commands
    if (command === "transparency") {
      await handleTransparencyCommand(subcommand, options);
    } else if (command === "help" || command === "--help" || command === "-h") {
      printHelp();
    } else if (command === "version" || command === "--version" || command === "-v") {
      console.log("transparency-service version 0.1.0");
    } else {
      error(`Unknown command: ${command}`);
      printHelp();
      process.exit(1);
    }
  } catch (err) {
    error(`Command failed: ${err}`);
    process.exit(1);
  }
}

/**
 * Handle transparency commands
 */
async function handleTransparencyCommand(subcommand: string | undefined, options: Record<string, any>) {
  if (!subcommand) {
    error("Transparency subcommand required");
    printTransparencyHelp();
    process.exit(1);
  }

  if (subcommand === "init") {
    const { init } = await import("./commands/transparency/init.ts");
    await init({
      database: options.database,
      storage: options.storage,
      port: options.port ? parseInt(options.port, 10) : undefined,
      force: options.force === true,
    });
  } else if (subcommand === "serve") {
    const { serve } = await import("./commands/transparency/serve.ts");
    await serve({
      port: options.port ? parseInt(options.port, 10) : undefined,
      hostname: options.hostname,
      database: options.database,
      config: options.config,
    });
  } else if (subcommand === "help" || subcommand === "--help") {
    printTransparencyHelp();
  } else {
    error(`Unknown transparency subcommand: ${subcommand}`);
    printTransparencyHelp();
    process.exit(1);
  }
}

/**
 * Print main help
 */
function printHelp() {
  console.log(`
Transparency Service CLI

Usage: transparency <command> [options]

Commands:
  transparency init          Initialize a new transparency service
  transparency serve         Start the transparency service
  help                       Show this help message
  version                    Show version information

Run 'transparency <command> --help' for more information on a command.
`);
}

/**
 * Print transparency command help
 */
function printTransparencyHelp() {
  console.log(`
Transparency Service Commands

Usage: transparency <subcommand> [options]

Subcommands:
  init                       Initialize service (database, storage, keys)
    --database <path>        Database file path (default: ./transparency.db)
    --storage <path>         Storage directory path (default: ./storage)
    --port <number>          Service port (default: 3000)
    --force                  Overwrite existing files

  serve                      Start the transparency service
    --port <number>          Override port from config
    --hostname <string>      Override hostname from config
    --database <path>        Override database path from config
    --config <path>          Custom config file path

Examples:
  transparency init
  transparency init --database ./data/transparency.db --port 8080
  transparency serve
  transparency serve --port 8080
`);
}

// Run CLI
main().catch((err) => {
  error(`Fatal error: ${err}`);
  process.exit(1);
});
