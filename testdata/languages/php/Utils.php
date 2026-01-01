<?php
/**
 * Utils module - tests utility functions and helpers.
 * Tests: static methods, closures, arrow functions.
 */

declare(strict_types=1);

namespace App;

/**
 * Utility functions class.
 */
class Utils
{
    /**
     * Process string data - called from main, should be reachable.
     *
     * @param string[] $items
     */
    public static function processData(array $items): string
    {
        if (empty($items)) {
            return '';
        }
        return implode(', ', array_map('strtoupper', $items));
    }

    /**
     * Format output string - called from main, should be reachable.
     */
    public static function formatOutput(string $data): string
    {
        return "Result: {$data}";
    }

    /**
     * Validate configuration - called from various places, should be reachable.
     */
    public static function validateConfig(Config $config): bool
    {
        return $config->validate();
    }

    // ========================================================================
    // Dead code section - functions that are never called
    // ========================================================================

    /**
     * Hash string - DEAD CODE.
     */
    public static function hashString(string $s): string
    {
        $hash = 5381;
        for ($i = 0; $i < strlen($s); $i++) {
            $hash = (($hash << 5) + $hash) + ord($s[$i]);
        }
        return sprintf('%016x', $hash & 0xFFFFFFFFFFFFFFFF);
    }

    /**
     * Filter strings - DEAD CODE.
     *
     * @param string[] $items
     * @param callable(string): bool $predicate
     * @return string[]
     */
    public static function filterStrings(array $items, callable $predicate): array
    {
        return array_values(array_filter($items, $predicate));
    }

    /**
     * Map strings - DEAD CODE.
     *
     * @param string[] $items
     * @param callable(string): string $transform
     * @return string[]
     */
    public static function mapStrings(array $items, callable $transform): array
    {
        return array_map($transform, $items);
    }

    /**
     * Reduce strings - DEAD CODE.
     *
     * @param string[] $items
     * @param callable(string, string): string $accumulator
     */
    public static function reduceStrings(array $items, string $seed, callable $accumulator): string
    {
        return array_reduce($items, $accumulator, $seed);
    }

    /**
     * Chunk array - DEAD CODE.
     *
     * @template T
     * @param T[] $items
     * @return T[][]
     */
    public static function chunkArray(array $items, int $size): array
    {
        return array_chunk($items, $size);
    }

    /**
     * Flatten nested arrays - DEAD CODE.
     *
     * @template T
     * @param T[][] $items
     * @return T[]
     */
    public static function flattenArray(array $items): array
    {
        return array_merge(...$items);
    }

    /**
     * Group by key - DEAD CODE.
     *
     * @template TKey
     * @template TValue
     * @param TValue[] $items
     * @param callable(TValue): TKey $keySelector
     * @return array<TKey, TValue[]>
     */
    public static function groupByKey(array $items, callable $keySelector): array
    {
        $result = [];
        foreach ($items as $item) {
            $key = $keySelector($item);
            $result[$key][] = $item;
        }
        return $result;
    }

    /**
     * Sort by comparator - DEAD CODE.
     *
     * @template T
     * @param T[] $items
     * @param callable(T, T): int $comparator
     * @return T[]
     */
    public static function sortBy(array $items, callable $comparator): array
    {
        usort($items, $comparator);
        return $items;
    }

    /**
     * Distinct by key - DEAD CODE.
     *
     * @template T
     * @template TKey
     * @param T[] $items
     * @param callable(T): TKey $keySelector
     * @return T[]
     */
    public static function distinctBy(array $items, callable $keySelector): array
    {
        $seen = [];
        $result = [];
        foreach ($items as $item) {
            $key = $keySelector($item);
            if (!isset($seen[$key])) {
                $seen[$key] = true;
                $result[] = $item;
            }
        }
        return $result;
    }

    /**
     * Take while predicate - DEAD CODE.
     *
     * @template T
     * @param T[] $items
     * @param callable(T): bool $predicate
     * @return T[]
     */
    public static function takeWhile(array $items, callable $predicate): array
    {
        $result = [];
        foreach ($items as $item) {
            if (!$predicate($item)) {
                break;
            }
            $result[] = $item;
        }
        return $result;
    }
}

/**
 * String helper class - DEAD CODE.
 */
class StringHelper
{
    /**
     * Check if string is blank - DEAD CODE.
     */
    public static function isBlank(?string $s): bool
    {
        return $s === null || trim($s) === '';
    }

    /**
     * Truncate string - DEAD CODE.
     */
    public static function truncate(string $s, int $maxLength): string
    {
        if (strlen($s) <= $maxLength) {
            return $s;
        }
        return substr($s, 0, $maxLength) . '...';
    }

    /**
     * Capitalize string - DEAD CODE.
     */
    public static function capitalize(string $s): string
    {
        return ucfirst(strtolower($s));
    }
}
