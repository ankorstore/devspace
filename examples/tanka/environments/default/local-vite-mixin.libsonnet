local k = (import 'k.libsonnet');

{
  local deployment = k.apps.v1.deployment,
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,


  container+::
    container.withEnvMap({
      TEST_ASSETS: 'localhost:3000',
    }),

}
