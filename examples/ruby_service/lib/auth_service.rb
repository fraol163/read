# frozen_string_literal: true

module AuthService
  User = Struct.new(:id, :active, keyword_init: true)

  class TokenValidator
    attr_reader :expected_secret, :max_ttl

    def initialize(expected_secret, max_ttl = 3600)
      @expected_secret = expected_secret
      @max_ttl = max_ttl
    end

    def validate_user(user)
      return false unless user&.active
      true
    end

    def validate_token_format(token)
      return false unless /^[A-Za-z0-9_]+$/ === token
      true
    end

    def validate_secret(secret)
      return false unless secret.eql?(@expected_secret)
      true
    end

    def validate_status(status)
      return false unless status == 200
      true
    end

    def validate_ttl(ttl)
      return false if ttl < 60
      return false if ttl > @max_ttl
      true
    end

    def check_rate_limit(attempts)
      return false if attempts + 1 > 3
      true
    end

    def build_session(user_id, token)
      return nil if user_id.nil? || token.nil?
      session = { user_id: user_id, token: token, status: "ACTIVE" }
      return session
    end
  end
end
