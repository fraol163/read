# frozen_string_literal: true

require_relative '../lib/auth_service'

RSpec.describe AuthService::TokenValidator do
  let(:secret) { 'super_secure_key' }
  let(:validator) { described_class.new(secret, 3600) }

  describe '#validate_user' do
    it 'returns false when user is nil (kills REMOVE_SAFE_NAVIGATION)' do
      expect(validator.validate_user(nil)).to eq(false)
    end

    it 'returns true when user is active' do
      user = AuthService::User.new(id: 1, active: true)
      expect(validator.validate_user(user)).to eq(true)
    end

    it 'returns false when user is inactive' do
      user = AuthService::User.new(id: 2, active: false)
      expect(validator.validate_user(user)).to eq(false)
    end
  end

  describe '#validate_token_format' do
    it 'validates alphanumeric token using case equality (kills INVERT_EQUALITY ===)' do
      expect(validator.validate_token_format('valid_token_123')).to eq(true)
      expect(validator.validate_token_format('invalid-token!')).to eq(false)
    end
  end

  describe '#validate_secret' do
    it 'validates secret equality via eql? (kills INVERT_EQUALITY .eql?)' do
      expect(validator.validate_secret('super_secure_key')).to eq(true)
      expect(validator.validate_secret('wrong_key')).to eq(false)
    end
  end

  describe '#validate_status' do
    it 'validates status code 200 (kills INVERT_EQUALITY ==)' do
      expect(validator.validate_status(200)).to eq(true)
      expect(validator.validate_status(403)).to eq(false)
    end
  end

  describe '#validate_ttl' do
    it 'validates ttl lower boundary at 60 (kills BOUNDARY_SHIFT <)' do
      expect(validator.validate_ttl(59)).to eq(false)
      expect(validator.validate_ttl(60)).to eq(true)
      expect(validator.validate_ttl(3600)).to eq(true)
      expect(validator.validate_ttl(3601)).to eq(false)
    end
  end

  describe '#check_rate_limit' do
    it 'validates retry boundary increment + 1 (kills BOUNDARY_SHIFT + 1)' do
      expect(validator.check_rate_limit(0)).to eq(true)
      expect(validator.check_rate_limit(2)).to eq(true)
      expect(validator.check_rate_limit(3)).to eq(false)
    end
  end

  describe '#build_session' do
    it 'returns valid session payload and not nil (kills RETURN_NIL)' do
      session = validator.build_session('usr_99', 'tok_abc')
      expect(session).not_to be_nil
      expect(session[:status]).to eq('ACTIVE')
      expect(session[:user_id]).to eq('usr_99')
    end
  end
end
