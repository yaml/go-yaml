# Copyright 2026 The go-yaml Project Contributors
# SPDX-License-Identifier: Apache-2.0

use strict;
use warnings;
use Test::More;
use Cwd qw(abs_path);
use File::Copy qw(copy);
use File::Path qw(make_path);
use File::Temp qw(tempdir);
use FindBin;
use JSON::PP qw(decode_json);

my $root = abs_path("$FindBin::Bin/..");
my $work = tempdir('release-test-XXXXXX', DIR => "$root/.cache", CLEANUP => 1);
make_path("$work/.cache", "$work/docs/releases");
copy("$root/GNUmakefile", "$work/GNUmakefile") or die "copy makefile: $!";
symlink("$root/.cache/makes", "$work/.cache/makes") or die "link Makes: $!";
copy("$root/docs/releases/v4.0.0.md", "$work/docs/releases/v4.0.0.md")
    or die "copy notes: $!";

my $mock = "$work/gh";
open my $fh, '>', $mock or die "write mock: $!";
print {$fh} '#!', abs_path($^X), "\n", <<'MOCK';
use strict;
use warnings;
use JSON::PP qw(encode_json);
open my $log, '>>', $ENV{GO_YAML_RELEASE_TEST_LOG} or die $!;
print {$log} encode_json(\@ARGV), "\n";
close $log or die $!;
if ($ENV{GO_YAML_RELEASE_TEST_STATUS}) {
    print STDERR "mock GitHub failure\n";
    exit $ENV{GO_YAML_RELEASE_TEST_STATUS};
}
print "https://github.com/yaml/go-yaml/releases/draft\n";
MOCK
close $fh or die "close mock: $!";
chmod 0755, $mock or die "chmod mock: $!";

sub run_make {
    my (@args) = @_;
    my $log = "$work/calls.jsonl";
    unlink $log if -e $log;
    local $ENV{GO_YAML_RELEASE_TEST_LOG} = $log;
    # Do not inherit parent make flags, including a test verbosity value.
    local $ENV{MAKEFLAGS} = '';
    local $ENV{MAKEOVERRIDES} = '';
    local $ENV{v};
    delete $ENV{v};
    my $pid = fork;
    die "fork: $!" unless defined $pid;
    if (!$pid) {
        open STDOUT, '>', "$work/output" or die "stdout: $!";
        open STDERR, '>&', STDOUT or die "stderr: $!";
        # -o prevents Makes from installing anything at the mock tool path.
        exec 'make', '--no-print-directory', '-C', $work,
            '-o', $mock, "GH=$mock", @args;
        die "exec make: $!";
    }
    waitpid($pid, 0);
    my $status = $?;
    open my $output, '<', "$work/output" or die "read output: $!";
    my $text = do { local $/; <$output> };
    my @calls;
    if (-e $log) {
        open my $calls, '<', $log or die "read calls: $!";
        @calls = map { decode_json($_) } <$calls>;
    }
    return ($status, $text, \@calls);
}

my ($status, $output, $calls) = run_make('release', 'v=4.0.0');
is $status, 0, 'release target succeeds with reviewed notes';
is_deeply $calls, [[
    'release', 'create', 'v4.0.0',
    '--repo', 'yaml/go-yaml', '--verify-tag', '--draft',
    '--title', 'go-yaml v4.0.0',
    '--notes-file', 'docs/releases/v4.0.0.md',
]], 'only creates a draft from an existing tag with the matching notes';
like $output, qr{https://github.com/yaml/go-yaml/releases/draft},
    'prints the draft URL';

for my $version (undef, '1', 'v4.0.0', '4.0.0-rc.9', '04.0.0',
    '4.0.0; touch injected', '4.0.0`touch injected`') {
    my @args = ('release');
    push @args, "v=$version" if defined $version;
    my ($status, $output, $calls) = run_make(@args);
    my $name = defined($version) ? $version : '(missing)';
    isnt $status, 0, "rejects version $name";
    like $output, qr/Usage: make release v=4\.0\.0/, 'explains version syntax';
    is_deeply $calls, [], 'does not invoke GitHub for an invalid version';
}
ok !-e "$work/injected", 'version input is not executed as shell source';

for my $empty (0, 1) {
    if ($empty) {
        open my $notes, '>', "$work/docs/releases/v9.0.0.md" or die $!;
        close $notes or die $!;
    }
    my ($status, $output, $calls) = run_make('release', 'v=9.0.0');
    isnt $status, 0, $empty ? 'rejects empty notes' : 'rejects missing notes';
    like $output, qr{Missing or empty release notes: docs/releases/v9\.0\.0\.md},
        'identifies the required notes file';
    is_deeply $calls, [], 'does not invoke GitHub without notes';
}

{
    local $ENV{GO_YAML_RELEASE_TEST_STATUS} = 7;
    my ($status, $output, $calls) = run_make('release', 'v=4.0.0');
    isnt $status, 0, 'propagates GitHub failure';
    like $output, qr/mock GitHub failure/, 'preserves GitHub diagnostics';
    is scalar(@$calls), 1, 'does not retry an unsuccessful create';
}

($status, $output, $calls) = run_make('-n', 'test-main', 'v=1');
is $status, 0, 'verbose test dry run succeeds';
like $output, qr/go test \.\s+-v/, 'v=1 still enables verbose tests';
is_deeply $calls, [], 'ordinary tests do not invoke GitHub';

done_testing;
