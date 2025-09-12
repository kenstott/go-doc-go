import yaml from 'js-yaml';

/**
 * Parse YAML string to JavaScript object
 */
export const parseYAML = (yamlString) => {
  try {
    return yaml.load(yamlString);
  } catch (error) {
    throw new Error(`YAML parsing error: ${error.message}`);
  }
};

/**
 * Convert JavaScript object to YAML string
 */
export const stringifyYAML = (obj, options = {}) => {
  try {
    return yaml.dump(obj, {
      indent: 2,
      lineWidth: -1, // No line wrapping
      noRefs: true, // Don't use references
      sortKeys: false, // Preserve key order
      ...options
    });
  } catch (error) {
    throw new Error(`YAML stringify error: ${error.message}`);
  }
};

/**
 * Validate YAML string
 */
export const validateYAML = (yamlString) => {
  try {
    yaml.load(yamlString);
    return { valid: true };
  } catch (error) {
    return {
      valid: false,
      error: error.message,
      line: error.mark?.line,
      column: error.mark?.column
    };
  }
};

/**
 * Pretty format YAML string
 */
export const formatYAML = (yamlString) => {
  try {
    const obj = yaml.load(yamlString);
    return yaml.dump(obj, {
      indent: 2,
      lineWidth: -1,
      noRefs: true,
      sortKeys: false
    });
  } catch (error) {
    throw new Error(`YAML formatting error: ${error.message}`);
  }
};

/**
 * Deep clone an object (useful for config editing)
 */
export const deepClone = (obj) => {
  return JSON.parse(JSON.stringify(obj));
};

/**
 * Get nested object value by path (e.g., 'storage.backend')
 */
export const getNestedValue = (obj, path) => {
  return path.split('.').reduce((current, key) => current?.[key], obj);
};

/**
 * Set nested object value by path (e.g., 'storage.backend', 'sqlite')
 */
export const setNestedValue = (obj, path, value) => {
  const keys = path.split('.');
  const lastKey = keys.pop();
  const target = keys.reduce((current, key) => {
    if (!(key in current)) current[key] = {};
    return current[key];
  }, obj);
  target[lastKey] = value;
  return obj;
};