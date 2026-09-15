import {
  Clash2UIF
} from "./clash2uif.js";

import {
  V2rayN2UIF
} from "./v2rayn2uif.js";

import {
  UIFRaw
} from "./uif_share.js";

import {
  Sing2UIF
} from "./sing2uif.js";

export default function TryParse(inputData) {
  console.log(inputData)
  const text = String(inputData || "").trim();
  const parsers = [V2rayN2UIF, UIFRaw, Sing2UIF, Clash2UIF];
  for (const parse of parsers) {
    try {
      const result = parse(text);
      if (Array.isArray(result) && result.length > 0) return result;
    } catch (error) {
      console.log("subscription parser failed", error);
    }
  }
  return [];
}
