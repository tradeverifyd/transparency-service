/**
 * CLI Output Formatting Utilities
 * Provides consistent output formatting for CLI commands
 */

/**
 * Output format type
 */
export type OutputFormat = "text" | "json";

/**
 * Print success message
 */
export function success(message: string): void {
  console.log(`✓ ${message}`);
}

/**
 * Print error message
 */
export function error(message: string): void {
  console.error(`✗ ${message}`);
}

/**
 * Print warning message
 */
export function warning(message: string): void {
  console.warn(`⚠ ${message}`);
}

/**
 * Print info message
 */
export function info(message: string): void {
  console.log(`ℹ ${message}`);
}

/**
 * Print section header
 */
export function header(title: string): void {
  console.log(`\n=== ${title} ===\n`);
}

/**
 * Print key-value pair
 */
export function keyValue(key: string, value: string | number | boolean): void {
  console.log(`  ${key}: ${value}`);
}

/**
 * Print object as formatted output
 */
export function printObject(obj: any, format: OutputFormat = "text"): void {
  if (format === "json") {
    console.log(JSON.stringify(obj, null, 2));
  } else {
    printTextObject(obj);
  }
}

/**
 * Print object in text format
 */
function printTextObject(obj: any, indent: number = 0): void {
  const prefix = "  ".repeat(indent);

  for (const [key, value] of Object.entries(obj)) {
    if (value === null || value === undefined) {
      console.log(`${prefix}${key}: (not set)`);
    } else if (typeof value === "object" && !Array.isArray(value)) {
      console.log(`${prefix}${key}:`);
      printTextObject(value, indent + 1);
    } else if (Array.isArray(value)) {
      console.log(`${prefix}${key}: [${value.length} items]`);
    } else {
      console.log(`${prefix}${key}: ${value}`);
    }
  }
}

/**
 * Print table
 */
export function printTable(headers: string[], rows: string[][]): void {
  // Calculate column widths
  const widths = headers.map((h, i) => {
    const maxRowWidth = Math.max(...rows.map((r) => String(r[i] || "").length));
    return Math.max(h.length, maxRowWidth);
  });

  // Print header
  const headerRow = headers.map((h, i) => h.padEnd(widths[i]!)).join(" | ");
  console.log(headerRow);
  console.log(widths.map((w) => "-".repeat(w)).join("-+-"));

  // Print rows
  for (const row of rows) {
    const rowStr = row.map((cell, i) => String(cell || "").padEnd(widths[i]!)).join(" | ");
    console.log(rowStr);
  }
}

/**
 * Print progress indicator
 */
export function progress(message: string): void {
  process.stdout.write(`${message}...`);
}

/**
 * Clear progress indicator and print result
 */
export function progressDone(success: boolean = true): void {
  console.log(success ? " ✓" : " ✗");
}

/**
 * Ask for user confirmation
 */
export async function confirm(question: string): Promise<boolean> {
  process.stdout.write(`${question} (y/n): `);

  for await (const line of console) {
    const answer = line.trim().toLowerCase();
    if (answer === "y" || answer === "yes") {
      return true;
    } else if (answer === "n" || answer === "no") {
      return false;
    } else {
      process.stdout.write("Please answer y or n: ");
    }
  }

  return false;
}

/**
 * Print command usage
 */
export function usage(command: string, description: string, options: { flag: string; description: string }[]): void {
  console.log(`\nUsage: ${command}\n`);
  console.log(`${description}\n`);

  if (options.length > 0) {
    console.log("Options:");
    for (const opt of options) {
      console.log(`  ${opt.flag.padEnd(20)} ${opt.description}`);
    }
    console.log();
  }
}
