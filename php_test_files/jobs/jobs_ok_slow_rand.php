<?php

/**
 * @var Goridge\RelayInterface $relay
 */

use Spiral\Goridge;
use Spiral\RoadRunner;
use Spiral\Goridge\StreamRelay;
use Spiral\Roadrunner\Jobs\Consumer;
use Spiral\RoadRunner\Jobs\Serializer\JsonSerializer;

ini_set('display_errors', 'stderr');
require dirname(__DIR__) . "/vendor/autoload.php";

$rr = new RoadRunner\Worker(new StreamRelay(\STDIN, \STDOUT));
$consumer = new Consumer($rr, new JsonSerializer);

while ($task = $consumer->waitTask()) {
    try {
        $val = random_int(0, 1000);
        if ($val > 995) {
            sleep(60);
        }

        $task->complete();
    } catch (\Throwable $e) {
        $rr->error((string)$e);
    }
}
