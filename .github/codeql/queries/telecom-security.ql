/**
 * @name Telecom-specific security vulnerabilities
 * @description Custom CodeQL queries for O-RAN and 5G security patterns
 * @kind problem
 * @problem.severity error
 * @security-severity 8.0
 * @tags security
 *       telecom
 *       o-ran
 *       5g
 */

import go

/**
 * Detect hardcoded IMSI/IMEI values
 */
class HardcodedIMSI extends DataFlow::Configuration {
  HardcodedIMSI() { this = "HardcodedIMSI" }
  
  override predicate isSource(DataFlow::Node source) {
    exists(string value |
      source.asExpr().(StringLit).getValue() = value and
      value.regexpMatch("[0-9]{15}") and
      (value.prefix(3) = "310" or value.prefix(3) = "311" or value.prefix(3) = "460")
    )
  }
  
  override predicate isSink(DataFlow::Node sink) {
    exists(CallExpr call |
      sink.asExpr() = call.getAnArgument() and
      (
        call.getTarget().hasQualifiedName("fmt", "Printf") or
        call.getTarget().hasQualifiedName("log", "Print") or
        call.getTarget().hasQualifiedName("encoding/json", "Marshal")
      )
    )
  }
}

/**
 * Detect unencrypted A1/O1 interface communications
 */
class UnencryptedORANInterface extends DataFlow::Configuration {
  UnencryptedORANInterface() { this = "UnencryptedORANInterface" }
  
  override predicate isSource(DataFlow::Node source) {
    exists(CallExpr call |
      source.asExpr() = call and
      call.getTarget().hasQualifiedName("net/http", "NewRequest") and
      not exists(CallExpr tlsCall |
        tlsCall.getTarget().hasQualifiedName("crypto/tls", _)
      )
    )
  }
  
  override predicate isSink(DataFlow::Node sink) {
    exists(CallExpr call |
      sink.asExpr() = call.getAnArgument() and
      call.getTarget().getName().matches("%Send%")
    )
  }
}

/**
 * Detect missing authentication on RIC interfaces
 */
class MissingRICAuthentication extends DataFlow::Configuration {
  MissingRICAuthentication() { this = "MissingRICAuthentication" }
  
  override predicate isSource(DataFlow::Node source) {
    exists(FuncDecl handler |
      handler.getName().matches("%Handler%") and
      source.asExpr() = handler.getBody() and
      not exists(CallExpr authCheck |
        authCheck.getTarget().getName().matches("%Auth%") or
        authCheck.getTarget().getName().matches("%Verify%") or
        authCheck.getTarget().getName().matches("%Validate%")
      )
    )
  }
  
  override predicate isSink(DataFlow::Node sink) {
    exists(CallExpr call |
      sink.asExpr() = call and
      call.getTarget().hasQualifiedName("net/http", "ListenAndServe")
    )
  }
}

/**
 * Detect insecure random number generation for crypto keys
 */
class InsecureRandomForCrypto extends DataFlow::Configuration {
  InsecureRandomForCrypto() { this = "InsecureRandomForCrypto" }
  
  override predicate isSource(DataFlow::Node source) {
    exists(CallExpr call |
      source.asExpr() = call and
      call.getTarget().hasQualifiedName("math/rand", _) and
      not call.getTarget().hasQualifiedName("crypto/rand", _)
    )
  }
  
  override predicate isSink(DataFlow::Node sink) {
    exists(CallExpr call |
      sink.asExpr() = call.getAnArgument() and
      (
        call.getTarget().getName().matches("%Key%") or
        call.getTarget().getName().matches("%Crypt%") or
        call.getTarget().getName().matches("%Cipher%")
      )
    )
  }
}

/**
 * Detect cleartext storage of network function credentials
 */
class CleartextNFCredentials extends DataFlow::Configuration {
  CleartextNFCredentials() { this = "CleartextNFCredentials" }
  
  override predicate isSource(DataFlow::Node source) {
    exists(Field f |
      source.asExpr() = f.getARead() and
      (
        f.getName().matches("%password%") or
        f.getName().matches("%secret%") or
        f.getName().matches("%token%") or
        f.getName().matches("%apikey%")
      )
    )
  }
  
  override predicate isSink(DataFlow::Node sink) {
    exists(CallExpr call |
      sink.asExpr() = call.getAnArgument() and
      (
        call.getTarget().hasQualifiedName("os", "WriteFile") or
        call.getTarget().hasQualifiedName("io", "WriteString") or
        call.getTarget().hasQualifiedName("encoding/json", "Marshal")
      ) and
      not exists(CallExpr encCall |
        encCall.getTarget().getName().matches("%Encrypt%") or
        encCall.getTarget().getName().matches("%Hash%")
      )
    )
  }
}

/**
 * Detect SQL injection in configuration management
 */
class ConfigSQLInjection extends DataFlow::Configuration {
  ConfigSQLInjection() { this = "ConfigSQLInjection" }
  
  override predicate isSource(DataFlow::Node source) {
    exists(CallExpr call |
      source.asExpr() = call and
      (
        call.getTarget().hasQualifiedName("net/http", "FormValue") or
        call.getTarget().hasQualifiedName("net/http", "Query") or
        call.getTarget().getName() = "Param"
      )
    )
  }
  
  override predicate isSink(DataFlow::Node sink) {
    exists(CallExpr call |
      sink.asExpr() = call.getAnArgument() and
      (
        call.getTarget().hasQualifiedName("database/sql", "Query") or
        call.getTarget().hasQualifiedName("database/sql", "Exec") or
        call.getTarget().getName().matches("%Query%")
      ) and
      not call.getTarget().getName().matches("%Prepared%")
    )
  }
}

/**
 * Detect missing rate limiting on API endpoints
 */
class MissingRateLimiting extends DataFlow::Configuration {
  MissingRateLimiting() { this = "MissingRateLimiting" }
  
  override predicate isSource(DataFlow::Node source) {
    exists(FuncDecl handler |
      handler.getName().matches("%Handler%") and
      source.asExpr() = handler and
      not exists(CallExpr rateLimit |
        rateLimit.getTarget().getName().matches("%RateLimit%") or
        rateLimit.getTarget().getName().matches("%Throttle%") or
        rateLimit.getTarget().getName().matches("%Limit%")
      )
    )
  }
  
  override predicate isSink(DataFlow::Node sink) {
    exists(CallExpr call |
      sink.asExpr() = call and
      (
        call.getTarget().hasQualifiedName("net/http", "Handle") or
        call.getTarget().hasQualifiedName("net/http", "HandleFunc")
      )
    )
  }
}

/**
 * Detect insecure deserialization of network messages
 */
class InsecureDeserialization extends DataFlow::Configuration {
  InsecureDeserialization() { this = "InsecureDeserialization" }
  
  override predicate isSource(DataFlow::Node source) {
    exists(CallExpr call |
      source.asExpr() = call and
      (
        call.getTarget().hasQualifiedName("net/http", "Body") or
        call.getTarget().hasQualifiedName("io", "ReadAll")
      )
    )
  }
  
  override predicate isSink(DataFlow::Node sink) {
    exists(CallExpr call |
      sink.asExpr() = call.getAnArgument() and
      (
        call.getTarget().hasQualifiedName("encoding/json", "Unmarshal") or
        call.getTarget().hasQualifiedName("encoding/xml", "Unmarshal") or
        call.getTarget().hasQualifiedName("encoding/gob", "Decode")
      ) and
      not exists(CallExpr validate |
        validate.getTarget().getName().matches("%Validate%") or
        validate.getTarget().getName().matches("%Sanitize%")
      )
    )
  }
}

from DataFlow::Configuration config, DataFlow::Node source, DataFlow::Node sink
where
  config.hasFlow(source, sink)
select sink.getLocation(), 
       "Security vulnerability: " + config.toString() + " flows from $@ to here.",
       source.getLocation(), "source"