local ssm = std.native('ssm');

{
  Architectures: [
    'arm64',
  ],
  EphemeralStorage: {
    Size: 512,
  },
  FunctionName: 'g3-scoring',
  Handler: 'bootstrap',
  LoggingConfig: {
    LogFormat: 'Text',
    LogGroup: '/aws/lambda/g3-scoring',
  },
  MemorySize: 512,
  Role: 'arn:aws:iam::601230306569:role/service-role/g3-scoring-role-bguxry7n',
  Runtime: 'provided.al2',
  SnapStart: {
    ApplyOn: 'None',
  },
  Timeout: 60,
  TracingConfig: {
    Mode: 'PassThrough',
  },
  VpcConfig: {
    SecurityGroupIds: [
      'sg-06513a902e2952f7c',
    ],
    SubnetIds: [
      'subnet-0dc2433c8f1fd92f2',
    ],
  },
  Environment: {
    Variables: {
      DB_HOST: ssm('/group3/database_host'),
      DB_USER: ssm('/group3/database_user'),
      DB_PASSWORD: ssm('/group3/database_password'),
      DB_NAME: ssm('/group3/database_name'),
    },
  },
}
